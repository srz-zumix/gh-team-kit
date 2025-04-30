package gh

import (
	"context"

	"github.com/google/go-github/v71/github"
	ghc "github.com/k1LoW/go-github-client/v71/factory"
)

type GitHubClient struct {
	client *github.Client
}

// NewGitHubClient creates a new GitHubClient instance using k1LoW/go-github-client
func NewGitHubClient() (*GitHubClient, error) {
	client, err := ghc.NewGithubClient()
	if err != nil {
		return nil, err
	}

	return &GitHubClient{
		client: client,
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
