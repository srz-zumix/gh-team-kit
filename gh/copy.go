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
				if existingPermission == permission {
					continue // Skip if the permission is already the same
				}
				return fmt.Errorf("team %s already has %s permissions on the destination repository", team.GetSlug(), existingPermission)
			}
		}

		if err := g.AddTeamRepo(ctx, src.Owner, team.GetSlug(), dst.Owner, dst.Name, permission); err != nil {
			return fmt.Errorf("failed to add team %s to destination repository: %w", team.GetSlug(), err)
		}
	}

	return nil
}

func SyncRepoTeamsAndPermissions(ctx context.Context, g *GitHubClient, src repository.Repository, dst repository.Repository) error {
	// Fetch teams and permissions from the source repository
	srcTeams, err := g.ListRepositoryTeams(ctx, src.Owner, src.Name)
	if err != nil {
		return fmt.Errorf("failed to fetch teams from source repository: %w", err)
	}

	// Fetch existing teams and permissions from the destination repository
	dstTeams, err := g.ListRepositoryTeams(ctx, dst.Owner, dst.Name)
	if err != nil {
		return fmt.Errorf("failed to fetch teams from destination repository: %w", err)
	}

	dstTeamMap := make(map[string]string)
	for _, team := range dstTeams {
		dstTeamMap[team.GetSlug()] = team.GetPermission()
	}

	srcTeamMap := make(map[string]string)
	for _, team := range srcTeams {
		srcTeamMap[team.GetSlug()] = team.GetPermission()
	}

	// Sync teams and permissions
	for _, team := range srcTeams {
		permission := team.GetPermission()

		if existingPermission, exists := dstTeamMap[team.GetSlug()]; exists {
			if existingPermission == permission {
				continue // Skip if the permission is already the same
			}
		}

		if err := g.AddTeamRepo(ctx, src.Owner, team.GetSlug(), dst.Owner, dst.Name, permission); err != nil {
			return fmt.Errorf("failed to sync team %s to destination repository: %w", team.GetSlug(), err)
		}
	}

	// Remove teams from the destination repository that are not in the source repository
	for _, team := range dstTeams {
		if _, exists := srcTeamMap[team.GetSlug()]; !exists {
			if err := g.RemoveTeamRepo(ctx, dst.Owner, team.GetSlug(), dst.Owner, dst.Name); err != nil {
				return fmt.Errorf("failed to remove team %s from destination repository: %w", team.GetSlug(), err)
			}
		}
	}

	return nil
}

// SyncTeamMembers syncs members from the source team to the destination team.
func SyncTeamMembers(ctx context.Context, g *GitHubClient, srcRepo repository.Repository, srcTeamSlug string, dstRepo repository.Repository, dstTeamSlug string) error {
	srcMembers, err := ListTeamMembers(ctx, g, srcRepo, srcTeamSlug, nil, false)
	if err != nil {
		return fmt.Errorf("failed to fetch members from source team: %w", err)
	}
	dstMembers, err := ListTeamMembers(ctx, g, dstRepo, dstTeamSlug, nil, false)
	if err != nil {
		return fmt.Errorf("failed to fetch members from destination team: %w", err)
	}
	dstMemberMap := make(map[string]struct{})
	for _, m := range dstMembers {
		if m.Login != nil {
			dstMemberMap[*m.Login] = struct{}{}
		}
	}
	srcMemberMap := make(map[string]struct{})
	for _, m := range srcMembers {
		if m.Login != nil {
			srcMemberMap[*m.Login] = struct{}{}
		}
	}
	// Add or update members in destination team
	for _, m := range srcMembers {
		if m.Login == nil {
			continue
		}
		username := *m.Login
		_, err := AddTeamMember(ctx, g, dstRepo, dstTeamSlug, username, "member", false)
		if err != nil {
			return fmt.Errorf("failed to add member %s to destination team: %w", username, err)
		}
	}
	// Remove members from destination team that are not in source team
	for _, m := range dstMembers {
		if m.Login == nil {
			continue
		}
		username := *m.Login
		if _, exists := srcMemberMap[username]; !exists {
			err := RemoveTeamMember(ctx, g, dstRepo, dstTeamSlug, username)
			if err != nil {
				return fmt.Errorf("failed to remove member %s from destination team: %w", username, err)
			}
		}
	}
	return nil
}
