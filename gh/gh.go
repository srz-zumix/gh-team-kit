package gh

import (
	"context"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

// ListTeams is a wrapper function that uses a Repository object to call either ListTeams or ListTeamsByRepo.
func ListTeams(ctx context.Context, g *GitHubClient, repo repository.Repository) ([]*github.Team, error) {
	if repo.Name != "" {
		return g.ListTeamsByRepo(ctx, repo.Owner, repo.Name)
	}
	return g.ListTeams(ctx, repo.Owner)
}
