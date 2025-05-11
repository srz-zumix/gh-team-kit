package client

import (
	"github.com/google/go-github/v71/github"
)

type GitHubClient struct {
	client *github.Client
}

func NewClient(client *github.Client) (*GitHubClient, error) {
	return &GitHubClient{
		client: client,
	}, nil
}

// GetClient returns the underlying GitHub client
func (g *GitHubClient) GetClient() *github.Client {
	return g.client
}
