package team

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewAddCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "add <team-slug> <org-role>",
		Short: "Add a team to an organization role",
		Long:  `Add a specified team to the specified role in the organization.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
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

			if err := gh.AssignOrgRoleToTeam(ctx, client, repository, teamSlug, orgRole); err != nil {
				return fmt.Errorf("failed to add team '%s' to role '%s' in organization '%s': %w", teamSlug, orgRole, owner, err)
			}

			fmt.Printf("Successfully added team '%s' to role '%s' in organization '%s'.\n", teamSlug, orgRole, owner)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")

	return cmd
}
