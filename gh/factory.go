package gh

import (
	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/k1LoW/go-github-client/v71/factory"
	"github.com/srz-zumix/gh-team-kit/gh/client"
)

type GitHubClient = client.GitHubClient

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
			c.Token, _ = auth.TokenForHost(host)
		}
		c.Owner = repo.Owner
		c.Repo = repo.Name
		return nil
	}
}

// NewGitHubClient creates a new GitHubClient instance using k1LoW/go-github-client
func NewGitHubClient() (*GitHubClient, error) {
	c, err := factory.NewGithubClient()
	if err != nil {
		return nil, err
	}

	return client.NewClient(c)
}

// NewGitHubClientWithRepo creates a new GitHubClient instance with a specified go-gh Repository.
func NewGitHubClientWithRepo(repo repository.Repository) (*GitHubClient, error) {
	c, err := factory.NewGithubClient(RepositoryOption(repo))
	if err != nil {
		return nil, err
	}
	return client.NewClient(c)
}

func NewGitHubClientWith2Repos(repo1, repo2 repository.Repository) (*GitHubClient, *GitHubClient, error) {
	c1, err := NewGitHubClientWithRepo(repo1)
	if err != nil {
		return nil, nil, err
	}
	if repo1.Host != repo2.Host {
		c2, err := NewGitHubClientWithRepo(repo2)
		if err != nil {
			return nil, nil, err
		}
		return c1, c2, nil
	}
	return c1, c1, nil
}
