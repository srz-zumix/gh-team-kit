package collaborator

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewRemoveCmd creates a new `user collaborator remove` command
func NewRemoveCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "remove <username>...",
		Short: "Remove outside collaborators from the organization",
		Long: `Remove the specified outside collaborators from all repositories of the organization.
Organization members cannot be removed with this command; use 'user remove' instead.`,
		Aliases: []string{"rm"},
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			var errs []error
			for _, username := range args {
				if err := gh.RemoveOutsideCollaborator(ctx, client, repository, username); err != nil {
					errs = append(errs, fmt.Errorf("failed to remove outside collaborator %s from organization %s: %w", username, repository.Owner, err))
					continue
				}
				logger.Info("Outside collaborator removed from organization successfully.", "username", username, "owner", repository.Owner)
			}
			return errors.Join(errs...)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")

	return cmd
}
