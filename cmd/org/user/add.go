package user

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

func NewAddCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "add <username> <org-role>",
		Short: "Assign a user to an organization role",
		Long:  `Assign a specified user to the specified role in the organization.`,
		Args:  cobra.ExactArgs(2),
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

			if err := gh.AssignOrgRoleToUser(ctx, client, repository, username, orgRole); err != nil {
				return fmt.Errorf("failed to assign user '%s' to role '%s' in organization '%s': %w", username, orgRole, owner, err)
			}

			fmt.Printf("Successfully assigned user '%s' to role '%s' in organization '%s'.\n", username, orgRole, owner)
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the organization")

	return cmd
}
