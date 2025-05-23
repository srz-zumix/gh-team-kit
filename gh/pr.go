package gh

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

func GetPullRequest(ctx context.Context, g *GitHubClient, repo repository.Repository, pull_request string) (*github.PullRequest, error) {
	number, err := GetPullRequestNumberFromString(pull_request)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pull request number from '%s': %w", pull_request, err)
	}
	pr, err := g.GetPullRequest(ctx, repo.Owner, repo.Name, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request #%d in repository '%s/%s': %w", number, repo.Owner, repo.Name, err)
	}
	return pr, nil
}

func GetPullRequestByNumber(ctx context.Context, g *GitHubClient, repo repository.Repository, number int) (*github.PullRequest, error) {
	pr, err := g.GetPullRequest(ctx, repo.Owner, repo.Name, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request #%d in repository '%s/%s': %w", number, repo.Owner, repo.Name, err)
	}
	return pr, nil
}

func GetPullRequestNumberFromString(pr string) (int, error) {
	number, err := GetNumberFromString(pr)
	if err != nil {
		return 0, err
	}
	return number, nil
}
