package client

import (
	"context"

	"github.com/google/go-github/v71/github"
)

// GetIssueByNumber retrieves an issue by owner, repo, and issue number
func (g *GitHubClient) GetIssueByNumber(ctx context.Context, owner, repo string, number int) (*github.Issue, error) {
	issue, _, err := g.client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	return issue, nil
}
