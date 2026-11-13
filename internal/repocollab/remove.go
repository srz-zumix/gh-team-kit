// Package repocollab provides reusable logic for repository collaborator operations.
package repocollab

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// BuildRemoveChanges builds removal changes for the given collaborators.
func BuildRemoveChanges(destination string, users []*github.User) []gh.SyncChange {
	changes := make([]gh.SyncChange, 0, len(users))
	for _, u := range users {
		changes = append(changes, gh.SyncChange{
			Destination: destination,
			Action:      gh.SyncChangeRemove,
			Target:      u.GetLogin(),
			Current:     gh.GetPermissionName(u.Permissions),
		})
	}
	return changes
}

// PlanRemoveSuspendedCollaborators lists direct collaborators of the repository,
// resolves their suspension state and returns removal changes for suspended users.
func PlanRemoveSuspendedCollaborators(ctx context.Context, client *gh.GitHubClient, repo repository.Repository) ([]gh.SyncChange, error) {
	collaborators, err := gh.ListRepositoryCollaborators(ctx, client, repo, []string{"direct"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list direct collaborators to find suspended users: %w", err)
	}
	collaborators, err = gh.UpdateUsersForSuspension(ctx, client, collaborators)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve suspension state of collaborators: %w", err)
	}
	return BuildRemoveChanges(parser.GetRepositoryFullNameWithHost(repo), gh.CollectSuspendedUsers(collaborators)), nil
}
