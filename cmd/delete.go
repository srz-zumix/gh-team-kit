package cmd

import (
	"context"
	"fmt"

	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"

	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "delete <team>",
		Short: "Delete a team",
		Long:  `Delete a specified team from the organization.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			team := args[0]

			ctx := context.Background()

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.DeleteTeam(ctx, client, repository, team); err != nil {
				return fmt.Errorf("failed to delete team: %w", err)
			}

			fmt.Printf("Team '%s' deleted successfully.\n", team)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization owner (optional)")

	return cmd
}

func init() {
	rootCmd.AddCommand(NewDeleteCmd())
}
