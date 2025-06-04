package user

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewRemoveCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:     "remove <username> <org-role>",
		Short:   "Remove a user from an organization role",
		Long:    `Remove a specified user from the specified role in the organization.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			orgRole := args[1]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository owner: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.RemoveOrgRoleFromUser(ctx, client, repository, username, orgRole); err != nil {
				return fmt.Errorf("failed to remove user '%s' from role '%s' in organization '%s': %w", username, orgRole, owner, err)
			}

			fmt.Printf("Successfully removed user '%s' from role '%s' in organization '%s'.\n", username, orgRole, owner)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "", "", "Specify the organization name")

	return cmd
}
