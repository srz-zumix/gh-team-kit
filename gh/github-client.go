package gh

import (
	"context"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
	"github.com/k1LoW/go-github-client/v71/factory"
)

type GitHubClient struct {
	client *github.Client
	IO     *iostreams.IOStreams
}

const defaultHost = "github.com"
const defaultV3Endpoint = "https://api.github.com"

func RepositoryOption(repo repository.Repository) factory.Option {
	return func(c *factory.Config) error {
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
	client, err := factory.NewGithubClient()
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
	client, err := factory.NewGithubClient(RepositoryOption(repo))
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

// ListChildTeams retrieves all child teams of a specified team.
func (g *GitHubClient) ListChildTeams(ctx context.Context, org string, parentSlug string) ([]*github.Team, error) {
	var allChildTeams []*github.Team
	opt := &github.ListOptions{PerPage: 50}

	for {
		teams, resp, err := g.client.Teams.ListChildTeamsByParentSlug(ctx, org, parentSlug, opt)
		if err != nil {
			return nil, err
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

// FindTeamBySlug retrieves a team by its slug name.
// If the team is not found, it returns nil without an error.
func (g *GitHubClient) FindTeamBySlug(ctx context.Context, org string, teamSlug string) (*github.Team, error) {
	t, resp, err := g.client.Teams.GetTeamBySlug(ctx, org, teamSlug)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

// ListTeamRepos retrieves all repositories associated with a specific team in the organization.
func (g *GitHubClient) ListTeamRepos(ctx context.Context, org string, teamSlug string) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opt := &github.ListOptions{PerPage: 50}

	for {
		repos, resp, err := g.client.Teams.ListTeamReposBySlug(ctx, org, teamSlug, opt)
		if err != nil {
			return nil, err
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
func (g *GitHubClient) CheckTeamPermissions(ctx context.Context, org string, teamSlug string, owner string, repo string) (*github.Repository, error) {
	teamRepo, resp, err := g.client.Teams.IsTeamRepoBySlug(ctx, org, teamSlug, owner, repo)
	if err != nil {
		if resp.StatusCode == 404 {
			return nil, nil
		}
		return nil, err
	}
	return teamRepo, nil
}

// RemoveTeamRepo removes a repository from a team in the organization.
func (g *GitHubClient) RemoveTeamRepo(ctx context.Context, org string, teamSlug string, owner string, repoName string) error {
	_, err := g.client.Teams.RemoveTeamRepoBySlug(ctx, org, teamSlug, owner, repoName)
	if err != nil {
		return err
	}
	return nil
}

// AddTeamRepo adds a repository to a team in the organization.
func (g *GitHubClient) AddTeamRepo(ctx context.Context, org string, teamSlug string, owner string, repoName string, permission string) error {
	opt := &github.TeamAddTeamRepoOptions{
		Permission: permission,
	}
	_, err := g.client.Teams.AddTeamRepoBySlug(ctx, org, teamSlug, owner, repoName, opt)
	if err != nil {
		return err
	}
	return nil
}

// AddOrUpdateTeamMembership adds or updates the membership of a user in a team.
func (g *GitHubClient) AddTeamMember(ctx context.Context, org string, teamSlug string, username string, role string) error {
	opt := &github.TeamAddTeamMembershipOptions{
		Role: role,
	}
	_, _, err := g.client.Teams.AddTeamMembershipBySlug(ctx, org, teamSlug, username, opt)
	if err != nil {
		return err
	}
	return nil
}

// RemoveTeamMember removes a user from a team in the organization.
func (g *GitHubClient) RemoveTeamMember(ctx context.Context, org string, teamSlug string, username string) error {
	_, err := g.client.Teams.RemoveTeamMembershipBySlug(ctx, org, teamSlug, username)
	if err != nil {
		return err
	}
	return nil
}

// ListTeamMembers retrieves all members of a specific team in the organization.
func (g *GitHubClient) ListTeamMembers(ctx context.Context, org string, teamSlug string, role string) ([]*github.User, error) {
	var allMembers []*github.User
	opt := &github.TeamListTeamMembersOptions{
		Role:        role,
		ListOptions: github.ListOptions{PerPage: 50},
	}

	for {
		members, resp, err := g.client.Teams.ListTeamMembersBySlug(ctx, org, teamSlug, opt)
		if err != nil {
			return nil, err
		}
		allMembers = append(allMembers, members...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allMembers, nil
}

// GetTeamMembership retrieves the membership details of a user in a specific team.
// If the user is not a member, it returns nil without an error.
func (g *GitHubClient) GetTeamMembership(ctx context.Context, org string, teamSlug string, username string) (*github.Membership, error) {
	membership, resp, err := g.client.Teams.GetTeamMembershipBySlug(ctx, org, teamSlug, username)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil // User is not a member
		}
		return nil, err
	}
	return membership, nil
}

// GetOrgMembership retrieves the membership details of a user in the organization.
func (g *GitHubClient) GetOrgMembership(ctx context.Context, owner string, username string) (*github.Membership, error) {
	membership, resp, err := g.client.Organizations.GetOrgMembership(ctx, username, owner)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil // User is not a member
		}
		return nil, err // Other errors
	}

	return membership, nil
}

// GetUser retrieves a user by their username.
func (g *GitHubClient) GetUser(ctx context.Context, username string) (*github.User, error) {
	user, _, err := g.client.Users.Get(ctx, username)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ListRepositoryTeams retrieves all teams associated with a specific repository.
func (g *GitHubClient) ListRepositoryTeams(ctx context.Context, owner string, repo string) ([]*github.Team, error) {
	var allTeams []*github.Team
	opt := &github.ListOptions{PerPage: 50}

	for {
		teams, resp, err := g.client.Repositories.ListTeams(ctx, owner, repo, opt)
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

// CreateTeam creates a new team in the specified organization.
func (g *GitHubClient) CreateTeam(ctx context.Context, org string, team *github.NewTeam) (*github.Team, error) {
	createdTeam, _, err := g.client.Teams.CreateTeam(ctx, org, *team)
	if err != nil {
		return nil, err
	}
	return createdTeam, nil
}

// DeleteTeamBySlug deletes a team by its slug in the specified organization.
func (g *GitHubClient) DeleteTeamBySlug(ctx context.Context, org string, teamSlug string) error {
	_, err := g.client.Teams.DeleteTeamBySlug(ctx, org, teamSlug)
	if err != nil {
		return err
	}
	return nil
}

func (g *GitHubClient) Write(exporter cmdutil.Exporter, data any) error {
	return exporter.Write(g.IO, data)
}
