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

// ListUserRepositories retrieves all repositories associated with a specific user.
func (g *GitHubClient) ListUserRepositories(ctx context.Context, username string, repoType string) ([]*github.Repository, error) {
	var allRepos []*github.Repository
	opt := &github.RepositoryListByUserOptions{
		Type: repoType,
		ListOptions: github.ListOptions{
			PerPage: 50,
		},
	}

	for {
		repos, resp, err := g.client.Repositories.ListByUser(ctx, username, opt)
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
