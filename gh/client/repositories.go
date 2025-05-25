package client

import (
	"context"

	"github.com/google/go-github/v71/github"
)

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

// ListOrganizationRepositories retrieves all repositories for a specific organization.
func (g *GitHubClient) ListOrganizationRepositories(ctx context.Context, org string, repoType string) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opt := &github.RepositoryListByOrgOptions{Type: repoType, ListOptions: github.ListOptions{PerPage: 50}}

	for {
		repos, resp, err := g.client.Repositories.ListByOrg(ctx, org, opt)
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

func (g *GitHubClient) CheckRepositoryCollaborators(ctx context.Context, owner string, repo string, username string) (bool, error) {
	collaborator, _, err := g.client.Repositories.IsCollaborator(ctx, owner, repo, username)
	if err != nil {
		return false, err
	}
	return collaborator, nil
}

// GetRepositoryPermission retrieves the permission level of a user for a specific repository.
func (g *GitHubClient) GetRepositoryPermission(ctx context.Context, owner string, repo string, username string) (*github.RepositoryPermissionLevel, error) {
	permission, resp, err := g.client.Repositories.GetPermissionLevel(ctx, owner, repo, username)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil // User not found or no permission
		}
		return nil, err
	}
	return permission, nil
}

// ListRepositoryCollaborators retrieves all collaborators for a specific repository.
func (g *GitHubClient) ListRepositoryCollaborators(ctx context.Context, owner string, repo string, affiliation string) ([]*github.User, error) {
	var allCollaborators []*github.User
	opt := &github.ListCollaboratorsOptions{
		Affiliation: affiliation,
		ListOptions: github.ListOptions{
			PerPage: 50,
		},
	}

	for {
		collaborators, resp, err := g.client.Repositories.ListCollaborators(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}
		allCollaborators = append(allCollaborators, collaborators...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allCollaborators, nil
}

// RemoveRepositoryCollaborator removes a collaborator from a specific repository.
func (g *GitHubClient) RemoveRepositoryCollaborator(ctx context.Context, owner string, repo string, username string) error {
	_, err := g.client.Repositories.RemoveCollaborator(ctx, owner, repo, username)
	return err
}

// AddRepositoryCollaborator adds a collaborator to a specific repository with a given permission.
func (g *GitHubClient) AddRepositoryCollaborator(ctx context.Context, owner string, repo string, username string, permission string) (*github.CollaboratorInvitation, error) {
	invitation, _, err := g.client.Repositories.AddCollaborator(ctx, owner, repo, username, &github.RepositoryAddCollaboratorOptions{Permission: permission})
	if err != nil {
		return nil, err
	}
	return invitation, nil
}
