package client

import (
	"context"

	"github.com/google/go-github/v71/github"
)

func (g *GitHubClient) GetPullRequest(ctx context.Context, owner string, repo string, number int) (*github.PullRequest, error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	return pr, nil
}
