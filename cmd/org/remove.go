package org

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewRemoveCmd creates a new `org remove` command.
func NewRemoveCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:     "remove <team-slug> <role>",
		Short:   "Remove a role from a team in the organization",
		Long:    `Remove a specified role from the specified team in the organization using the provided team slug and role name.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			role := args[1]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("failed to parse owner: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			if err := gh.RemoveOrgRoleFromTeam(ctx, client, repository, teamSlug, role); err != nil {
				return fmt.Errorf("failed to remove role '%s' from team '%s': %w", role, teamSlug, err)
			}

			fmt.Printf("Successfully removed role '%s' from team '%s'.\n", role, teamSlug)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "", "", "Specify the organization name")

	return cmd
}
