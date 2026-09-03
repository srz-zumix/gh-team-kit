package prgraph

import (
	"context"
	"fmt"
	"math"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/hmarr/codeowners"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// Weight bases selecting how a changed path contributes to path-derived edge weights.
const (
	WeightByOccurrences = "occurrences"
	WeightByLines       = "lines"
	WeightByAdditions   = "additions"
	WeightByDeletions   = "deletions"
)

// WeightByValues lists all valid weight bases in display order.
var WeightByValues = []string{
	WeightByOccurrences,
	WeightByLines,
	WeightByAdditions,
	WeightByDeletions,
}

// Options controls which pull requests are analyzed.
type Options struct {
	State string     // Pull request state filter: "open", "closed", "merged", or "all"
	Since *time.Time // Only include pull requests created at or after this time
	Until *time.Time // Only include pull requests created at or before this time
	Limit int        // Maximum number of pull requests per repository (0 = unlimited)
	// ExcludeUsers is deprecated: it applies both ExcludeAuthors and HideUsers
	// filtering for backward compatibility. Prefer ExcludeAuthors and/or HideUsers.
	ExcludeUsers        []string
	ExcludeAuthors      []string // Skip pull requests authored by these users (PR selection filter)
	HideUsers           []string // Omit these users' non-author nodes and edges (rendering filter)
	ExcludeFiles        []string // .gitignore-style patterns for files to exclude from the graph
	IncludeFiles        []string // .gitignore-style patterns; when non-empty, only matching files are included
	Labels              []string // Only include pull requests having at least one of these labels
	ExcludeLabels       []string // Skip pull requests having any of these labels
	Base                string   // Only include pull requests whose base branch matches this glob pattern
	Head                string   // Only include pull requests whose head branch matches this glob pattern
	ExcludeHeadBranches []string // Skip pull requests whose head branch matches any of these glob patterns
	IncludeEdgeTypes    []string // Keep only edges of these relation types in the final graph
	ExcludeEdgeTypes    []string // Remove edges of these relation types from the final graph
	MinWeight           float64  // Remove edges with a weight below this threshold from the final graph (0 = no filter)
	NoBots              bool     // Automatically exclude/hide users whose login has a "[bot]" suffix
	ExcludeDraft        bool     // Skip draft pull requests
	// Depth folds a changed file's path into its ancestor directory truncated
	// to this many path segments (0 = no folding). Applied to paths that match
	// no GroupBy pattern.
	Depth int
	// GroupBy folds a changed file's path using glob-style prefix patterns
	// (e.g. "LocalPackages/*"), evaluated in order. Patterns take precedence
	// over Depth.
	GroupBy []string
	// ExcludeGenerated skips files marked with the linguist-generated attribute
	// in the repository's .gitattributes.
	ExcludeGenerated bool
	// KeepOrphans retains nodes left with no edges after edge filtering.
	KeepOrphans bool
	// WeightBy selects how a changed path contributes to path-derived edge
	// weights: WeightByOccurrences (default), WeightByLines, WeightByAdditions,
	// or WeightByDeletions. Relations that have no line count, such as reviews
	// and comments, always count occurrences.
	WeightBy string
	// HalfLife decays every contribution by the age of its pull request,
	// halving it every HalfLife days (0 = no decay).
	HalfLife float64
	// CoChange adds co-changed edges between the paths of the same pull request.
	CoChange bool
	// CoChangeMaxFiles skips co-change edges for pull requests touching more
	// than this many distinct paths, bounding the quadratic pair count
	// (0 = unlimited).
	CoChangeMaxFiles int
	// AllowUsers restricts the users kept in the graph to these logins. An
	// empty list keeps every user.
	AllowUsers []string
	// ExcludeDeleted skips paths that no longer exist on the repository's
	// default branch.
	ExcludeDeleted bool
}

type pullRequestActivity struct {
	number             int
	author             string
	created            time.Time
	labels             []string
	requestedReviewers []string
	requestedTeams     []string
}

// pathContribution accumulates the weight a single pull request contributes to
// one path node, used to build co-change pairs.
type pathContribution struct {
	node   *Node
	weight float64
}

// collector accumulates PR activity into a graph.
type collector struct {
	client *gh.GitHubClient
	graph  *Graph
	opts   Options
	// involvedUsers tracks users that appeared in PR activity per organization owner
	involvedUsers map[string]map[string]bool
	multiRepo     bool
	// multiOwner is true when the analysis spans more than one repository owner,
	// requiring team nodes to be namespaced to avoid cross-owner slug collisions.
	multiOwner     bool
	excludeFiles   *ignore.GitIgnore
	includeFiles   *ignore.GitIgnore
	excludeAuthors []string
	hideUsers      []string
	allowUsers     []string
	// referenceTime is the timestamp contributions are aged against when
	// --half-life is set. It is resolved once so that a run is reproducible.
	referenceTime time.Time
	// submodules holds the .gitmodules paths of the repository being collected.
	submodules map[string]bool
	// generatedFiles matches the linguist-generated patterns of the repository
	// being collected.
	generatedFiles *ignore.GitIgnore
	// trackedPaths holds the tracked paths (blobs and submodule commit entries)
	// of the default branch of the repository being collected. It is nil unless
	// --exclude-deleted resolved a complete tree.
	trackedPaths map[string]bool
}

// Collect analyzes pull request activity in the given repositories and builds
// a relationship graph of users, teams, labels, files, and directories.
func Collect(ctx context.Context, client *gh.GitHubClient, repos []repository.Repository, opts Options) (*Graph, error) {
	c := &collector{
		client:         client,
		graph:          NewGraph(),
		opts:           opts,
		involvedUsers:  make(map[string]map[string]bool),
		multiRepo:      len(repos) > 1,
		multiOwner:     distinctOwners(repos) > 1,
		excludeFiles:   ignore.CompileIgnoreLines(opts.ExcludeFiles...),
		includeFiles:   compileIncludeFiles(opts.IncludeFiles),
		excludeAuthors: combineLogins(opts.ExcludeAuthors, opts.ExcludeUsers),
		hideUsers:      combineLogins(opts.HideUsers, opts.ExcludeUsers),
		allowUsers:     opts.AllowUsers,
		referenceTime:  decayReference(opts),
	}
	for _, repo := range repos {
		if err := c.collectRepository(ctx, repo); err != nil {
			return nil, err
		}
	}
	c.collectTeamMemberships(ctx, repos)
	c.graph.RoundWeights()
	c.graph.FilterEdges(opts.IncludeEdgeTypes, opts.ExcludeEdgeTypes)
	c.graph.FilterMinWeight(opts.MinWeight)
	if !opts.KeepOrphans {
		c.graph.RemoveOrphanNodes()
	}
	return c.graph, nil
}

// decayReference returns the timestamp contributions are aged against. It uses
// the --until bound when set so that repeating the same command yields the same
// weights, and the current time otherwise.
func decayReference(opts Options) time.Time {
	if opts.Until != nil {
		return *opts.Until
	}
	return time.Now()
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
	case "closed", "merged":
		listOpts = append(listOpts, gh.ListPullRequestsOptionStateClosed())
	default:
		listOpts = append(listOpts, gh.ListPullRequestsOptionStateAll())
	}

	// Stops paginating once a page is entirely older than --since (when set),
	// instead of fetching the repository's entire pull request history.
	prs, err := gh.ListPullRequestsSince(ctx, c.client, repo, c.opts.Since, listOpts...)
	if err != nil {
		return fmt.Errorf("failed to list pull requests for %s/%s: %w", repo.Owner, repo.Name, err)
	}

	ruleset := fetchCodeowners(ctx, c.client, repo)
	c.submodules = fetchSubmodulePaths(ctx, c.client, repo)
	c.generatedFiles = nil
	if c.opts.ExcludeGenerated {
		c.generatedFiles = fetchGeneratedMatcher(ctx, c.client, repo)
	}
	c.trackedPaths = nil
	if c.opts.ExcludeDeleted {
		c.trackedPaths = fetchTrackedPaths(ctx, c.client, repo)
	}

	count := 0
	for _, pr := range prs {
		created := pr.GetCreatedAt().Time
		// PRs are sorted by creation date descending, so once a PR is older
		// than --since all remaining PRs are older too and can be skipped.
		if c.opts.Since != nil && created.Before(*c.opts.Since) {
			break
		}
		if c.opts.Until != nil && created.After(*c.opts.Until) {
			continue
		}
		if c.opts.State == "merged" && pr.GetMergedAt().IsZero() {
			continue
		}
		if c.opts.ExcludeDraft && pr.GetDraft() {
			continue
		}
		var labels []string
		for _, label := range pr.Labels {
			if name := label.GetName(); name != "" {
				labels = append(labels, name)
			}
		}
		if !matchesLabelFilters(labels, c.opts.Labels, c.opts.ExcludeLabels) {
			continue
		}
		if !matchesBranchPattern(c.opts.Base, pr.GetBase().GetRef()) {
			continue
		}
		if !matchesBranchPattern(c.opts.Head, pr.GetHead().GetRef()) {
			continue
		}
		if matchesAnyBranchPattern(pr.GetHead().GetRef(), c.opts.ExcludeHeadBranches) {
			continue
		}
		if c.isExcludedAuthor(pr.GetUser().GetLogin()) {
			continue
		}
		if c.opts.Limit > 0 && count >= c.opts.Limit {
			break
		}
		count++
		activity := pullRequestActivity{
			number:  pr.GetNumber(),
			author:  pr.GetUser().GetLogin(),
			created: created,
			labels:  labels,
		}
		for _, user := range pr.RequestedReviewers {
			if login := user.GetLogin(); login != "" {
				activity.requestedReviewers = append(activity.requestedReviewers, login)
			}
		}
		for _, team := range pr.RequestedTeams {
			if slug := team.GetSlug(); slug != "" {
				activity.requestedTeams = append(activity.requestedTeams, slug)
			}
		}
		if err := c.collectPullRequest(ctx, repo, pr, activity, ruleset); err != nil {
			return err
		}
	}
	fields := []any{"repository", repo.Owner + "/" + repo.Name, "fetched", len(prs), "analyzed", count}
	if c.opts.Since != nil {
		fields = append(fields, "since", c.opts.Since.Format(time.RFC3339))
	}
	if c.opts.Until != nil {
		fields = append(fields, "until", c.opts.Until.Format(time.RFC3339))
	}
	logger.Info("collected pull request activity", fields...)
	return nil
}

// collectPullRequest adds nodes and edges derived from a single pull request.
// An author outside the allowlist yields no author node: the pull request is
// still analyzed so that its paths keep contributing directory, CODEOWNERS and
// co-change edges, but no user-anchored edge is created for it.
func (c *collector) collectPullRequest(ctx context.Context, repo repository.Repository, pr any, activity pullRequestActivity, ruleset codeowners.Ruleset) error {
	author := activity.author
	if author == "" || c.isExcludedAuthor(author) {
		return nil
	}
	var authorNode *Node
	if c.isAllowedUser(author) {
		authorNode = c.addUser(repo, author)
	}
	decay := c.decayFactor(activity.created)

	if authorNode != nil {
		if err := c.collectParticipants(ctx, repo, pr, activity, authorNode, decay); err != nil {
			return err
		}
	}
	return c.collectPaths(ctx, repo, pr, activity, ruleset, authorNode, decay)
}

// collectParticipants adds the label, review request, review and comment edges
// that point at the pull request author.
func (c *collector) collectParticipants(ctx context.Context, repo repository.Repository, pr any, activity pullRequestActivity, authorNode *Node, decay float64) error {
	author := activity.author

	// Labels: author -> label
	for _, name := range activity.labels {
		labelNode := c.graph.AddNode(NodeTypeLabel, name)
		c.graph.AddEdgeWeight(authorNode, labelNode, RelationLabeled, decay)
	}

	// Requested reviewers: reviewer/team -> author
	for _, login := range activity.requestedReviewers {
		if login != author {
			c.addUserEdge(repo, login, authorNode, RelationReviewRequested, decay)
		}
	}
	for _, slug := range activity.requestedTeams {
		teamNode := c.graph.AddNode(NodeTypeTeam, c.teamName(repo.Owner, slug))
		c.graph.AddEdgeWeight(teamNode, authorNode, RelationReviewRequested, decay)
	}

	// Reviews: reviewer -> author, relation by review state
	reviews, err := gh.GetPullRequestReviews(ctx, c.client, repo, pr)
	if err != nil {
		return fmt.Errorf("failed to get reviews for %s/%s#%d: %w", repo.Owner, repo.Name, activity.number, err)
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
		c.addUserEdge(repo, login, authorNode, relation, decay)
	}

	// Review comments (comments on code): commenter -> author
	reviewComments, err := gh.ListPullRequestReviewComments(ctx, c.client, repo, pr)
	if err != nil {
		return fmt.Errorf("failed to list review comments for %s/%s#%d: %w", repo.Owner, repo.Name, activity.number, err)
	}
	for _, comment := range reviewComments {
		login := comment.GetUser().GetLogin()
		if login == "" || login == author {
			continue
		}
		c.addUserEdge(repo, login, authorNode, RelationReviewCommented, decay)
	}

	// Issue comments (conversation comments): commenter -> author
	issueComments, err := gh.ListIssueComments(ctx, c.client, repo, activity.number)
	if err != nil {
		return fmt.Errorf("failed to list comments for %s/%s#%d: %w", repo.Owner, repo.Name, activity.number, err)
	}
	for _, comment := range issueComments {
		login := comment.GetUser().GetLogin()
		if login == "" || login == author {
			continue
		}
		c.addUserEdge(repo, login, authorNode, RelationCommented, decay)
	}

	return nil
}

// collectPaths adds the edges derived from the pull request's changed files:
// author -> path, the directory chain, CODEOWNERS ownership, and the co-change
// pairs of the pull request.
func (c *collector) collectPaths(ctx context.Context, repo repository.Repository, pr any, activity pullRequestActivity, ruleset codeowners.Ruleset, authorNode *Node, decay float64) error {
	files, err := gh.ListPullRequestFiles(ctx, c.client, repo, pr)
	if err != nil {
		return fmt.Errorf("failed to list files for %s/%s#%d: %w", repo.Owner, repo.Name, activity.number, err)
	}
	contributions := make(map[string]*pathContribution)
	var order []string
	for _, file := range files {
		filename := file.GetFilename()
		if filename == "" || c.isExcludedFile(filename) || !c.isIncludedFile(filename) || c.isGeneratedFile(filename) || c.isDeletedFile(filename) {
			continue
		}
		// A zero contribution, such as a deletion-only file weighted by
		// additions, carries no information and is left out of the graph.
		weight := c.fileWeight(file.GetAdditions(), file.GetDeletions()) * decay
		if weight <= 0 {
			continue
		}
		groupedPath, folded := c.groupedPath(filename)
		fileNode := c.graph.AddNode(c.pathNodeType(groupedPath, folded), c.pathName(repo, groupedPath))
		if authorNode != nil {
			c.graph.AddEdgeWeight(authorNode, fileNode, RelationChanged, weight)
			if isCreatedStatus(file.GetStatus()) {
				c.graph.AddEdgeWeight(authorNode, fileNode, RelationCreated, weight)
			}
		}
		c.addDirectoryChain(repo, fileNode, groupedPath, weight)
		c.addCodeownersEdges(repo, fileNode, filename, ruleset, weight)
		if c.opts.CoChange {
			if existing, ok := contributions[fileNode.ID]; ok {
				existing.weight += weight
			} else {
				contributions[fileNode.ID] = &pathContribution{node: fileNode, weight: weight}
				order = append(order, fileNode.ID)
			}
		}
	}
	c.addCoChangeEdges(activity.number, contributions, order)
	return nil
}

// addCoChangeEdges links every pair of distinct paths touched by the same pull
// request. The relation is undirected, so each pair is stored once with the
// lexicographically smaller node ID as its source. The pair weight is the
// smaller of the two contributions, which keeps a single sweeping change from
// dominating every pair it touches.
func (c *collector) addCoChangeEdges(number int, contributions map[string]*pathContribution, order []string) {
	if !c.opts.CoChange || len(contributions) < 2 {
		return
	}
	// The pair count grows quadratically, so wide pull requests such as
	// repository-wide reformatting are skipped instead of exploding the graph.
	if c.opts.CoChangeMaxFiles > 0 && len(contributions) > c.opts.CoChangeMaxFiles {
		logger.Info("skipped co-change edges",
			"pull_request", number,
			"paths", len(contributions),
			"max", c.opts.CoChangeMaxFiles,
		)
		return
	}
	ids := append([]string(nil), order...)
	sort.Strings(ids)
	for i, fromID := range ids {
		from := contributions[fromID]
		for _, toID := range ids[i+1:] {
			to := contributions[toID]
			c.graph.AddEdgeWeight(from.node, to.node, RelationCoChanged, math.Min(from.weight, to.weight))
		}
	}
}

// isCreatedStatus reports whether a pull request file status means the author
// introduced the path. Renames are excluded because the content already existed
// under its previous name.
func isCreatedStatus(status string) bool {
	return status == "added" || status == "copied"
}

// fileWeight returns the contribution of one changed file according to
// --weight-by. Line-based bases let a rewrite outweigh a one-line fix, while
// the default counts every changed file once.
func (c *collector) fileWeight(additions, deletions int) float64 {
	switch c.opts.WeightBy {
	case WeightByLines:
		return float64(additions + deletions)
	case WeightByAdditions:
		return float64(additions)
	case WeightByDeletions:
		return float64(deletions)
	default:
		return 1
	}
}

// decayFactor returns the exponential decay applied to a contribution made at
// the given time, halving it every --half-life days. It is 1 when decay is
// disabled, the time is unknown, or the contribution is not older than the
// reference time.
func (c *collector) decayFactor(created time.Time) float64 {
	if c.opts.HalfLife <= 0 || created.IsZero() {
		return 1
	}
	ageDays := c.referenceTime.Sub(created).Hours() / 24
	if ageDays <= 0 {
		return 1
	}
	return math.Pow(0.5, ageDays/c.opts.HalfLife)
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

// addUserEdge adds an edge from a user unless that user is hidden.
func (c *collector) addUserEdge(repo repository.Repository, login string, to *Node, relation string, weight float64) {
	if c.isHiddenUser(login) {
		return
	}
	c.graph.AddEdgeWeight(c.addUser(repo, login), to, relation, weight)
}

// isExcludedAuthor reports whether login matches an --exclude-author
// (or deprecated --exclude-user) pattern, or --no-bots is set and login has
// a "[bot]" suffix; matching pull requests are skipped entirely.
func (c *collector) isExcludedAuthor(login string) bool {
	return matchesAnyLogin(login, c.excludeAuthors) || (c.opts.NoBots && isBotLogin(login))
}

// isHiddenUser reports whether login matches a --hide-user (or deprecated
// --exclude-user) pattern, --no-bots is set and login has a "[bot]"
// suffix, or an allowlist is configured and login is not on it; matching
// users are omitted from the graph as non-author participants, without
// affecting pull request selection.
func (c *collector) isHiddenUser(login string) bool {
	return matchesAnyLogin(login, c.hideUsers) || (c.opts.NoBots && isBotLogin(login)) || !c.isAllowedUser(login)
}

// isAllowedUser reports whether login may appear in the graph. Without an
// allowlist every user is allowed. Unlike --hide-user, a login rejected here is
// also dropped as a pull request author, while its pull request still
// contributes path-derived edges.
func (c *collector) isAllowedUser(login string) bool {
	return len(c.allowUsers) == 0 || matchesAnyLogin(login, c.allowUsers)
}

// isBotLogin reports whether login follows the GitHub App bot convention of
// ending with a "[bot]" suffix (e.g. "dependabot[bot]").
func isBotLogin(login string) bool {
	return strings.HasSuffix(strings.ToLower(login), "[bot]")
}

func matchesAnyLogin(login string, logins []string) bool {
	for _, l := range logins {
		if strings.EqualFold(login, l) {
			return true
		}
	}
	return false
}

// combineLogins merges primary with the deprecated list, avoiding an
// allocation when the deprecated list is empty.
func combineLogins(primary, deprecated []string) []string {
	if len(deprecated) == 0 {
		return primary
	}
	return append(append([]string{}, primary...), deprecated...)
}

// matchesLabelFilters reports whether labels satisfy the --label (include)
// and --exclude-label filters. An empty include list matches everything.
func matchesLabelFilters(labels, include, exclude []string) bool {
	if len(exclude) > 0 && anyLabelMatches(labels, exclude) {
		return false
	}
	if len(include) > 0 && !anyLabelMatches(labels, include) {
		return false
	}
	return true
}

func anyLabelMatches(labels, filters []string) bool {
	for _, label := range labels {
		for _, filter := range filters {
			if strings.EqualFold(label, filter) {
				return true
			}
		}
	}
	return false
}

// matchesBranchPattern reports whether branch matches the given glob pattern
// (as accepted by path.Match). An empty pattern matches every branch.
func matchesBranchPattern(pattern, branch string) bool {
	if pattern == "" {
		return true
	}
	ok, err := path.Match(pattern, branch)
	return err == nil && ok
}

// matchesAnyBranchPattern reports whether branch matches any of the given glob patterns.
func matchesAnyBranchPattern(branch string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesBranchPattern(pattern, branch) {
			return true
		}
	}
	return false
}

// isGeneratedFile reports whether filename is marked linguist-generated in the
// repository's .gitattributes. It is always false unless --exclude-generated is set.
func (c *collector) isGeneratedFile(filename string) bool {
	return c.generatedFiles != nil && c.generatedFiles.MatchesPath(filename)
}

// isExcludedFile reports whether filename matches a .gitignore-style
// --exclude-file pattern. A nil matcher (no patterns configured) excludes nothing.
func (c *collector) isExcludedFile(filename string) bool {
	return c.excludeFiles != nil && c.excludeFiles.MatchesPath(filename)
}

// isDeletedFile reports whether filename is absent from the default branch of
// the repository. It is always false unless --exclude-deleted resolved a
// complete tree.
func (c *collector) isDeletedFile(filename string) bool {
	return c.trackedPaths != nil && !c.trackedPaths[filename]
}

// compileIncludeFiles compiles --include-file patterns, returning nil when
// no patterns are configured so that isIncludedFile allows every file.
func compileIncludeFiles(patterns []string) *ignore.GitIgnore {
	if len(patterns) == 0 {
		return nil
	}
	return ignore.CompileIgnoreLines(patterns...)
}

// isIncludedFile reports whether filename matches a .gitignore-style
// --include-file pattern. A nil matcher (no patterns configured) includes everything.
func (c *collector) isIncludedFile(filename string) bool {
	return c.includeFiles == nil || c.includeFiles.MatchesPath(filename)
}

// groupedPath folds filename into a directory/package-level path using the
// first matching --group-by pattern, falling back to --depth when no pattern
// matches. It reports whether the path was folded. Without either option set,
// filename is returned unchanged.
func (c *collector) groupedPath(filename string) (string, bool) {
	for _, pattern := range c.opts.GroupBy {
		if grouped, ok := groupByPattern(filename, pattern); ok {
			return grouped, grouped != filename
		}
	}
	if c.opts.Depth > 0 {
		grouped := truncatePathDepth(filename, c.opts.Depth)
		return grouped, grouped != filename
	}
	return filename, false
}

// pathNodeType classifies a changed path node. A folded path denotes a
// directory rather than the changed file itself, while an unfolded path that
// matches a .gitmodules entry is a submodule pointer update.
func (c *collector) pathNodeType(p string, folded bool) NodeType {
	switch {
	case folded:
		return NodeTypeDirectory
	case c.submodules[p]:
		return NodeTypeSubmodule
	default:
		return NodeTypeFile
	}
}

// truncatePathDepth returns the first depth path segments of filename,
// joined by "/". filename is returned unchanged if it has depth segments or fewer.
func truncatePathDepth(filename string, depth int) string {
	segments := strings.Split(filename, "/")
	if depth >= len(segments) {
		return filename
	}
	return strings.Join(segments[:depth], "/")
}

// groupByPattern matches pattern against filename segment by segment, where
// a "*" pattern segment matches any single filename segment. On a match, it
// returns the filename truncated to the pattern's length.
func groupByPattern(filename, pattern string) (string, bool) {
	fileSegments := strings.Split(filename, "/")
	patternSegments := strings.Split(pattern, "/")
	if len(patternSegments) > len(fileSegments) {
		return "", false
	}
	for i, segment := range patternSegments {
		if segment != "*" && segment != fileSegments[i] {
			return "", false
		}
	}
	return strings.Join(fileSegments[:len(patternSegments)], "/"), true
}

// pathName returns the node name for a repository path, prefixed with the
// repository when multiple repositories are analyzed.
func (c *collector) pathName(repo repository.Repository, p string) string {
	if c.multiRepo {
		return fmt.Sprintf("%s/%s:%s", repo.Owner, repo.Name, p)
	}
	return p
}

// teamName returns the team node name. When the analysis spans multiple owners,
// the slug is namespaced with its organization so that same-slug teams in
// different organizations do not collide into a single node.
func (c *collector) teamName(org, slug string) string {
	if c.multiOwner {
		return org + "/" + slug
	}
	return slug
}

// distinctOwners counts the distinct owners (per host) among the repositories.
func distinctOwners(repos []repository.Repository) int {
	owners := make(map[string]bool)
	for _, repo := range repos {
		owners[repo.Host+"/"+repo.Owner] = true
	}
	return len(owners)
}

// addDirectoryChain links a file node to its parent directories up to the repository root.
func (c *collector) addDirectoryChain(repo repository.Repository, fileNode *Node, filename string, weight float64) {
	child := fileNode
	for dir := path.Dir(filename); dir != "." && dir != "/"; dir = path.Dir(dir) {
		dirNode := c.graph.AddNode(NodeTypeDirectory, c.pathName(repo, dir))
		c.graph.AddEdgeWeight(child, dirNode, RelationInDirectory, weight)
		child = dirNode
	}
}

// addCodeownersEdges links a file node to its CODEOWNERS owners.
func (c *collector) addCodeownersEdges(repo repository.Repository, fileNode *Node, filename string, ruleset codeowners.Ruleset, weight float64) {
	if ruleset == nil {
		return
	}
	rule, err := ruleset.Match(filename)
	if err != nil || rule == nil {
		return
	}
	for _, owner := range rule.Owners {
		if owner.Type != codeowners.TeamOwner && c.isHiddenUser(owner.Value) {
			continue
		}
		ownerNode := codeownersOwnerNode(c.graph, repo, owner, c.multiOwner)
		c.graph.AddEdgeWeight(fileNode, ownerNode, RelationOwnedBy, weight)
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
		logins := make([]string, 0, len(c.involvedUsers[key]))
		for login := range c.involvedUsers[key] {
			logins = append(logins, login)
		}
		// Sort logins so node/edge insertion order is deterministic across runs.
		sort.Strings(logins)
		for _, login := range logins {
			teams, err := gh.ListUserTeams(ctx, c.client, repo, login)
			if err != nil {
				continue // skip users whose teams cannot be listed
			}
			userNode := c.graph.AddNode(NodeTypeUser, login)
			slugs := make([]string, 0, len(teams))
			for _, team := range teams {
				if slug := team.GetSlug(); slug != "" {
					slugs = append(slugs, slug)
				}
			}
			sort.Strings(slugs)
			for _, slug := range slugs {
				teamNode := c.graph.AddNode(NodeTypeTeam, c.teamName(repo.Owner, slug))
				c.graph.AddEdge(userNode, teamNode, RelationMemberOf)
			}
		}
	}
}
