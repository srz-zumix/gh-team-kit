package collaborator

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewConvertCmd creates a new `user collaborator convert` command
func NewConvertCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "convert <username>...",
		Short: "Convert organization members to outside collaborators",
		Long: `Convert the specified organization members to outside collaborators.
The converted users keep access only to the repositories that their current team membership allows, and are no longer members of the organization.`,
		Args: cobra.MinimumNArgs(1),
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
				if err := gh.ConvertMemberToOutsideCollaborator(ctx, client, repository, username); err != nil {
					errs = append(errs, fmt.Errorf("failed to convert %s to an outside collaborator of organization %s: %w", username, repository.Owner, err))
					continue
				}
				logger.Info("Member converted to outside collaborator successfully.", "username", username, "owner", repository.Owner)
			}
			return errors.Join(errs...)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")

	return cmd
}
