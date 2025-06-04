package member

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
		Use:     "remove <team-slug> <username>",
		Short:   "Remove a member from a team",
		Long:    `Remove a specified user from the specified team in the organization.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			username := args[1]
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.RemoveTeamMember(ctx, client, repository, teamSlug, username); err != nil {
				return fmt.Errorf("failed to remove member from team: %w", err)
			}

			fmt.Printf("Successfully removed user '%s' from team '%s'.\n", username, teamSlug)
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "", "", "Specify the organization name")

	return cmd
}
