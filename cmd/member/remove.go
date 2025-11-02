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

			err = gh.RemoveTeamMembers(ctx, client, repository, teamSlug, usernames)
			if err != nil {
				return fmt.Errorf("failed to remove member from team: %w", err)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")

	return cmd
}
