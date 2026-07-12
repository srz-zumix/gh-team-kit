package prgraph

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v88/github"
	"github.com/hmarr/codeowners"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// Options controls which pull requests are analyzed.
type Options struct {
	State string     // Pull request state filter: "open", "closed", or "all"
	Since *time.Time // Only include pull requests created at or after this time
	Until *time.Time // Only include pull requests created at or before this time
	Limit int        // Maximum number of pull requests per repository (0 = unlimited)
}

// collector accumulates PR activity into a graph.
type collector struct {
	client *gh.GitHubClient
	graph  *Graph
	opts   Options
	// involvedUsers tracks users that appeared in PR activity per organization owner
	involvedUsers map[string]map[string]bool
	multiRepo     bool
}

// Collect analyzes pull request activity in the given repositories and builds
// a relationship graph of users, teams, labels, files, and directories.
func Collect(ctx context.Context, client *gh.GitHubClient, repos []repository.Repository, opts Options) (*Graph, error) {
	c := &collector{
		client:        client,
		graph:         NewGraph(),
		opts:          opts,
		involvedUsers: make(map[string]map[string]bool),
		multiRepo:     len(repos) > 1,
	}
	for _, repo := range repos {
		if err := c.collectRepository(ctx, repo); err != nil {
			return nil, err
		}
	}
	c.collectTeamMemberships(ctx, repos)
	return c.graph, nil
}

// collectRepository analyzes the pull requests of a single repository.
func (c *collector) collectRepository(ctx context.Context, repo repository.Repository) error {
	listOpts := []gh.ListPullRequestsOption{
		gh.ListPullRequestsOptionSortCreated(),
		gh.ListPullRequestsOptionDirectionDescending(),
	}
	switch c.opts.State {
	case "open":
		listOpts = append(listOpts, gh.ListPullRequestsOptionStateOpen())
	case "closed":
		listOpts = append(listOpts, gh.ListPullRequestsOptionStateClosed())
	default:
		listOpts = append(listOpts, gh.ListPullRequestsOptionStateAll())
	}

	prs, err := gh.ListPullRequests(ctx, c.client, repo, listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list pull requests for %s/%s: %w", repo.Owner, repo.Name, err)
	}

	ruleset := fetchCodeowners(ctx, c.client, repo)

	count := 0
	for _, pr := range prs {
		created := pr.GetCreatedAt().Time
		if c.opts.Since != nil && created.Before(*c.opts.Since) {
			continue
		}
		if c.opts.Until != nil && created.After(*c.opts.Until) {
			continue
		}
		if c.opts.Limit > 0 && count >= c.opts.Limit {
			break
		}
		count++
		if err := c.collectPullRequest(ctx, repo, pr, ruleset); err != nil {
			return err
		}
	}
	return nil
}

// collectPullRequest adds nodes and edges derived from a single pull request.
func (c *collector) collectPullRequest(ctx context.Context, repo repository.Repository, pr *github.PullRequest, ruleset codeowners.Ruleset) error {
	author := pr.GetUser().GetLogin()
	if author == "" {
		return nil
	}
	authorNode := c.addUser(repo, author)

	// Labels: author -> label
	for _, label := range pr.Labels {
		if name := label.GetName(); name != "" {
			labelNode := c.graph.AddNode(NodeTypeLabel, name)
			c.graph.AddEdge(authorNode, labelNode, RelationLabeled)
		}
	}

	// Requested reviewers: reviewer/team -> author
	for _, user := range pr.RequestedReviewers {
		if login := user.GetLogin(); login != "" && login != author {
			c.graph.AddEdge(c.addUser(repo, login), authorNode, RelationReviewRequested)
		}
	}
	for _, team := range pr.RequestedTeams {
		if slug := team.GetSlug(); slug != "" {
			teamNode := c.graph.AddNode(NodeTypeTeam, slug)
			c.graph.AddEdge(teamNode, authorNode, RelationReviewRequested)
		}
	}

	// Reviews: reviewer -> author, relation by review state
	reviews, err := gh.GetPullRequestReviews(ctx, c.client, repo, pr)
	if err != nil {
		return fmt.Errorf("failed to get reviews for %s/%s#%d: %w", repo.Owner, repo.Name, pr.GetNumber(), err)
	}
	for _, review := range reviews {
		login := review.GetUser().GetLogin()
		if login == "" || login == author {
			continue
		}
		relation := RelationReviewed
		switch review.GetState() {
		case gh.PullRequestReviewStateApproved:
			relation = RelationApproved
		case gh.PullRequestReviewStateChangesRequested:
			relation = RelationChangesRequested
		}
		c.graph.AddEdge(c.addUser(repo, login), authorNode, relation)
	}

	// Review comments (comments on code): commenter -> author
	reviewComments, err := gh.ListPullRequestReviewComments(ctx, c.client, repo, pr)
	if err != nil {
		return fmt.Errorf("failed to list review comments for %s/%s#%d: %w", repo.Owner, repo.Name, pr.GetNumber(), err)
	}
	for _, comment := range reviewComments {
		login := comment.GetUser().GetLogin()
		if login == "" || login == author {
			continue
		}
		c.graph.AddEdge(c.addUser(repo, login), authorNode, RelationReviewCommented)
	}

	// Issue comments (conversation comments): commenter -> author
	issueComments, err := gh.ListIssueComments(ctx, c.client, repo, pr.GetNumber())
	if err != nil {
		return fmt.Errorf("failed to list comments for %s/%s#%d: %w", repo.Owner, repo.Name, pr.GetNumber(), err)
	}
	for _, comment := range issueComments {
		login := comment.GetUser().GetLogin()
		if login == "" || login == author {
			continue
		}
		c.graph.AddEdge(c.addUser(repo, login), authorNode, RelationCommented)
	}

	// Changed files: author -> file, file -> directory chain, file -> CODEOWNERS owner
	files, err := gh.ListPullRequestFiles(ctx, c.client, repo, pr)
	if err != nil {
		return fmt.Errorf("failed to list files for %s/%s#%d: %w", repo.Owner, repo.Name, pr.GetNumber(), err)
	}
	for _, file := range files {
		filename := file.GetFilename()
		if filename == "" {
			continue
		}
		fileNode := c.graph.AddNode(NodeTypeFile, c.pathName(repo, filename))
		c.graph.AddEdge(authorNode, fileNode, RelationChanged)
		c.addDirectoryChain(repo, fileNode, filename)
		c.addCodeownersEdges(repo, fileNode, filename, ruleset)
	}

	return nil
}

// addUser adds a user node and records the user as involved for team detection.
func (c *collector) addUser(repo repository.Repository, login string) *Node {
	key := repo.Host + "/" + repo.Owner
	if c.involvedUsers[key] == nil {
		c.involvedUsers[key] = make(map[string]bool)
	}
	c.involvedUsers[key][login] = true
	return c.graph.AddNode(NodeTypeUser, login)
}

// pathName returns the node name for a repository path, prefixed with the
// repository when multiple repositories are analyzed.
func (c *collector) pathName(repo repository.Repository, p string) string {
	if c.multiRepo {
		return fmt.Sprintf("%s/%s:%s", repo.Owner, repo.Name, p)
	}
	return p
}

// addDirectoryChain links a file node to its parent directories up to the repository root.
func (c *collector) addDirectoryChain(repo repository.Repository, fileNode *Node, filename string) {
	child := fileNode
	for dir := path.Dir(filename); dir != "." && dir != "/"; dir = path.Dir(dir) {
		dirNode := c.graph.AddNode(NodeTypeDirectory, c.pathName(repo, dir))
		c.graph.AddEdge(child, dirNode, RelationInDirectory)
		child = dirNode
	}
}

// addCodeownersEdges links a file node to its CODEOWNERS owners.
func (c *collector) addCodeownersEdges(repo repository.Repository, fileNode *Node, filename string, ruleset codeowners.Ruleset) {
	if ruleset == nil {
		return
	}
	rule, err := ruleset.Match(filename)
	if err != nil || rule == nil {
		return
	}
	for _, owner := range rule.Owners {
		ownerNode := codeownersOwnerNode(c.graph, repo, owner)
		c.graph.AddEdge(fileNode, ownerNode, RelationOwnedBy)
	}
}

// collectTeamMemberships adds user -> team membership edges for all involved users.
// Team lookup failures (e.g. insufficient permissions) are skipped.
func (c *collector) collectTeamMemberships(ctx context.Context, repos []repository.Repository) {
	seen := make(map[string]bool)
	for _, repo := range repos {
		key := repo.Host + "/" + repo.Owner
		if seen[key] {
			continue
		}
		seen[key] = true
		for login := range c.involvedUsers[key] {
			teams, err := gh.ListUserTeams(ctx, c.client, repo, login)
			if err != nil {
				continue // skip users whose teams cannot be listed
			}
			userNode := c.graph.AddNode(NodeTypeUser, login)
			for _, team := range teams {
				if slug := team.GetSlug(); slug != "" {
					teamNode := c.graph.AddNode(NodeTypeTeam, slug)
					c.graph.AddEdge(userNode, teamNode, RelationMemberOf)
				}
			}
		}
	}
}
