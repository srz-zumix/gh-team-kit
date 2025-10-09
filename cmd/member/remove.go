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
		Use:     "remove <team-slug> <username...>",
		Short:   "Remove a member from a team",
		Long:    `Remove a specified user from the specified team in the organization.`,
		Aliases: []string{"rm"},
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			usernames := args[1:]
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			var errors []error
			for _, username := range usernames {
				if err := gh.RemoveTeamMember(ctx, client, repository, teamSlug, username); err != nil {
					fmt.Printf("failed to remove member from team '%s': %v\n", teamSlug, err)
					errors = append(errors, err)
				} else {
					fmt.Printf("Successfully removed user '%s' from team '%s'.\n", username, teamSlug)
				}
			}

			if len(errors) > 0 {
				return fmt.Errorf("failed to remove %d user(s) from organization", len(errors))
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")

	return cmd
}
