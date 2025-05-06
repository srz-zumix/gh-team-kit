package gh

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// CopyRepoTeamsAndPermissions copies teams and permissions from the source repository to the destination repository.
func CopyRepoTeamsAndPermissions(ctx context.Context, g *GitHubClient, src repository.Repository, dst repository.Repository, force bool) error {
	// Fetch teams and permissions from the source repository
	srcTeams, err := g.ListRepositoryTeams(ctx, src.Owner, src.Name)
	if err != nil {
		return fmt.Errorf("failed to fetch teams from source repository: %w", err)
	}

	// Iterate over each team and copy permissions to the destination repository
	for _, team := range srcTeams {
		permission := team.GetPermission()

		// Check if the team already has permissions on the destination repository
		if !force {
			existingRepo, err := g.CheckTeamPermissions(ctx, dst.Owner, team.GetSlug(), dst.Owner, dst.Name)
			if err != nil {
				return fmt.Errorf("failed to check existing permissions for team %s: %w", team.GetSlug(), err)
			}
			if existingRepo != nil {
				existingPermission := GetRepositoryPermissions(existingRepo)
				return fmt.Errorf("team %s already has %s permissions on the destination repository", team.GetSlug(), existingPermission)
			}
		}

		if err := g.AddTeamRepo(ctx, src.Owner, team.GetSlug(), dst.Owner, dst.Name, permission); err != nil {
			return fmt.Errorf("failed to add team %s to destination repository: %w", team.GetSlug(), err)
		}
	}

	return nil
}
