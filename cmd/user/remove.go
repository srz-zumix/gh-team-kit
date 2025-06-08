package user

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewRemoveCmd creates a new remove command.
func NewRemoveCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:     "remove <username>",
		Short:   "Remove a user from the organization",
		Long:    `Remove a specified user from the organization using the provided username and optional owner information.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("failed to parse owner: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			if err = gh.RemoveOrgMember(ctx, client, repository, username); err != nil {
				return fmt.Errorf("failed to remove organization membership: %w", err)
			}
			fmt.Printf("Successfully removed user '%s' from the organization.\n", username)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Owner of the organization (optional)")

	return cmd
}
