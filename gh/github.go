package gh

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
	ghc "github.com/k1LoW/go-github-client/v71/factory"
)

type GitHubClient struct {
	client *github.Client
	IO     *iostreams.IOStreams
}

const defaultHost = "github.com"
const defaultV3Endpoint = "https://api.github.com"

func RepositoryOption(repo repository.Repository) ghc.Option {
	return func(c *ghc.Config) error {
		host := repo.Host
		if host != "" {
			if host == defaultHost {
				c.Endpoint = defaultV3Endpoint
			} else {
				c.Endpoint = "https://" + host + "/api/v3"
			}
		}
		c.Owner = repo.Owner
		c.Repo = repo.Name
		return nil
	}
}

// NewGitHubClient creates a new GitHubClient instance using k1LoW/go-github-client
func NewGitHubClient() (*GitHubClient, error) {
	client, err := ghc.NewGithubClient()
	if err != nil {
		return nil, err
	}

	return &GitHubClient{
		client: client,
		IO:     iostreams.System(),
	}, nil
}

// NewGitHubClientWithRepo creates a new GitHubClient instance with a specified go-gh Repository.
func NewGitHubClientWithRepo(repo repository.Repository) (*GitHubClient, error) {
	client, err := ghc.NewGithubClient(RepositoryOption(repo))
	if err != nil {
		return nil, err
	}

	return &GitHubClient{
		client: client,
		IO:     iostreams.System(),
	}, nil
}

// ListTeams retrieves all teams in the specified organization with pagination support
func (g *GitHubClient) ListTeams(ctx context.Context, org string) ([]*github.Team, error) {
	var allTeams []*github.Team
	opt := &github.ListOptions{PerPage: 50}

	for {
		teams, resp, err := g.client.Teams.ListTeams(ctx, org, opt)
		if err != nil {
			return nil, err
		}
		allTeams = append(allTeams, teams...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allTeams, nil
}

// ListTeamsByRepo retrieves all teams associated with a specific repository in the organization
func (g *GitHubClient) ListTeamsByRepo(ctx context.Context, org, repo string) ([]*github.Team, error) {
	var allTeams []*github.Team
	opt := &github.ListOptions{PerPage: 50}

	for {
		teams, resp, err := g.client.Teams.ListTeams(ctx, org, opt)
		if err != nil {
			return nil, err
		}
		for _, team := range teams {
			repos, _, err := g.client.Teams.ListTeamReposBySlug(ctx, org, team.GetSlug(), opt)
			if err != nil {
				return nil, err
			}
			for _, r := range repos {
				if r.GetName() == repo {
					allTeams = append(allTeams, team)
					break
				}
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allTeams, nil
}

// ListChildTeams retrieves all child teams of a specified team.
func (g *GitHubClient) ListChildTeams(ctx context.Context, org string, parentSlug string) ([]*github.Team, error) {
	var allChildTeams []*github.Team
	opt := &github.ListOptions{PerPage: 50}

	for {
		teams, resp, err := g.client.Teams.ListChildTeamsByParentSlug(ctx, org, parentSlug, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list child teams: %w", err)
		}
		allChildTeams = append(allChildTeams, teams...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allChildTeams, nil
}

// GetTeamBySlug retrieves a team by its slug name.
func (g *GitHubClient) GetTeamBySlug(ctx context.Context, org string, teamSlug string) (*github.Team, error) {
	team, _, err := g.client.Teams.GetTeamBySlug(ctx, org, teamSlug)
	if err != nil {
		return nil, err
	}
	return team, nil
}

// ListTeamRepos retrieves all repositories associated with a specific team in the organization.
func (g *GitHubClient) ListTeamRepos(ctx context.Context, org string, teamSlug string) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opt := &github.ListOptions{PerPage: 50}

	for {
		repos, resp, err := g.client.Teams.ListTeamReposBySlug(ctx, org, teamSlug, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list team repositories: %w", err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

// CheckTeamPermissions checks the permissions of a team for a specific repository.
func (g *GitHubClient) CheckTeamPermissions(ctx context.Context, org string, teamSlug string, repo string) (*github.Repository, error) {
	teamRepo, resp, err := g.client.Teams.IsTeamRepoBySlug(ctx, org, teamSlug, org, repo)
	if err != nil {
		if resp.StatusCode == 404 {
			return nil, nil
		}
		return nil, err
	}
	return teamRepo, nil
}

// RemoveTeamRepo removes a repository from a team in the organization.
func (g *GitHubClient) RemoveTeamRepo(ctx context.Context, org string, teamSlug string, repoName string) error {
	_, err := g.client.Teams.RemoveTeamRepoBySlug(ctx, org, teamSlug, org, repoName)
	if err != nil {
		return fmt.Errorf("failed to remove repository '%s' from team '%s': %w", repoName, teamSlug, err)
	}
	return nil
}

// AddTeamRepo adds a repository to a team in the organization.
func (g *GitHubClient) AddTeamRepo(ctx context.Context, org string, teamSlug string, repoName string, permission string) error {
	opt := &github.TeamAddTeamRepoOptions{
		Permission: permission,
	}
	_, err := g.client.Teams.AddTeamRepoBySlug(ctx, org, teamSlug, org, repoName, opt)
	if err != nil {
		return fmt.Errorf("failed to add repository '%s' to team '%s': %w", repoName, teamSlug, err)
	}
	return nil
}

func (g *GitHubClient) Write(exporter cmdutil.Exporter, data interface{}) error {
	return exporter.Write(g.IO, data)
}
