package user

import (
	"errors"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewRemoveCmd creates a new `repo user remove` command
func NewRemoveCmd() *cobra.Command {
	var dryrun bool
	var exporter cmdutil.Exporter
	var repo string
	var suspended bool

	cmd := &cobra.Command{
		Use:   "remove [<username>]",
		Short: "Remove a collaborator from a repository",
		Long: `Remove a specified user as a collaborator from the specified repository.
With --suspended, remove all suspended direct collaborators instead of a specified user.
With --dryrun, display the planned removals without applying them.`,
		Aliases: []string{"rm"},
		Args: func(cmd *cobra.Command, args []string) error {
			if suspended {
				if len(args) != 0 {
					return fmt.Errorf("<username> cannot be specified with --suspended")
				}
				return nil
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			destination := parser.GetRepositoryFullNameWithHost(repository)
			var changes []gh.SyncChange
			if suspended {
				collaborators, err := gh.ListRepositoryCollaborators(ctx, client, repository, []string{"direct"}, nil)
				if err != nil {
					return fmt.Errorf("failed to list direct collaborators to find suspended users: %w", err)
				}
				collaborators, err = gh.UpdateUsersForSuspension(ctx, client, collaborators)
				if err != nil {
					return fmt.Errorf("failed to resolve suspension state of collaborators: %w", err)
				}
				for _, u := range gh.CollectSuspendedUsers(collaborators) {
					changes = append(changes, gh.SyncChange{
						Destination: destination,
						Action:      gh.SyncChangeRemove,
						Target:      u.GetLogin(),
						Current:     gh.GetPermissionName(u.Permissions),
					})
				}
			} else {
				changes = append(changes, gh.SyncChange{
					Destination: destination,
					Action:      gh.SyncChangeRemove,
					Target:      args[0],
				})
			}

			if dryrun {
				if err := render.NewRenderer(exporter).RenderSyncChanges(changes, nil); err != nil {
					return fmt.Errorf("failed to render planned collaborator removals: %w", err)
				}
				return nil
			}

			if suspended && len(changes) == 0 {
				logger.Info("No suspended collaborators found.", "repository", parser.GetRepositoryFullName(repository))
				return nil
			}

			var errs []error
			for _, change := range changes {
				if err := gh.RemoveRepositoryCollaborator(ctx, client, repository, change.Target); err != nil {
					errs = append(errs, fmt.Errorf("failed to remove collaborator %s from repository: %w", change.Target, err))
					continue
				}
				logger.Info("User removed from repository successfully.", "username", change.Target, "repository", parser.GetRepositoryFullName(repository))
			}
			return errors.Join(errs...)
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: show planned removals without applying them")
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format '[HOST/]OWNER/REPO'")
	f.BoolVar(&suspended, "suspended", false, "Remove all suspended direct collaborators instead of a specified user")
	cmdutil.AddFormatFlags(cmd, &exporter)

	return cmd
}
