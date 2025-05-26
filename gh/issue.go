package gh

import (
	"context"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

func GetIssue(ctx context.Context, g *GitHubClient, repo repository.Repository, issue string) (*github.Issue, error) {
	number, err := GetIssueNumberFromString(issue)
	if err != nil {
		return nil, err
	}
	return g.GetIssueByNumber(ctx, repo.Owner, repo.Name, number)
}

// GetIssueByNumber retrieves an issue by repo (owner/repo) and issue number
func GetIssueByNumber(ctx context.Context, g *GitHubClient, repo repository.Repository, number int) (*github.Issue, error) {
	return g.GetIssueByNumber(ctx, repo.Owner, repo.Name, number)
}

func GetIssueNumberFromString(issue string) (int, error) {
	number, err := GetNumberFromString(issue)
	if err != nil {
		return 0, err
	}
	return number, nil
}
