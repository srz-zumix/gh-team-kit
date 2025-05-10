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

// GetRepositoryPermission retrieves the permission level of a user for a specific repository.
func (g *GitHubClient) GetRepositoryPermission(ctx context.Context, owner string, repo string, username string) (*github.RepositoryPermissionLevel, error) {
	permission, _, err := g.client.Repositories.GetPermissionLevel(ctx, owner, repo, username)
	if err != nil {
		return nil, err
	}
	return permission, nil
}
