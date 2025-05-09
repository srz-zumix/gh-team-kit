package client

import (
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/go-github/v71/github"
)

type GitHubClient struct {
	client *github.Client
	IO     *iostreams.IOStreams
}

func NewClient(client *github.Client, io *iostreams.IOStreams) (*GitHubClient, error) {
	return &GitHubClient{
		client: client,
		IO:     io,
	}, nil
}

// GetClient returns the underlying GitHub client
func (g *GitHubClient) GetClient() *github.Client {
	return g.client
}
