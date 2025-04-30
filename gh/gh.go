package gh

import (
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

// // ExampleMethod demonstrates a method that interacts with the GitHub API
// func (g *GitHubClient) ExampleMethod(ctx context.Context, owner, repo string) (interface{}, error) {
// 	repoInfo, err := g.client.Repository(ctx, owner, repo)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return repoInfo, nil
// }
