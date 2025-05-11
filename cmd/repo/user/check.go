package user

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type CheckOptions struct {
	Exporter cmdutil.Exporter
}

// NewCheckCmd creates a new `user check` command
func NewCheckCmd() *cobra.Command {
	opts := &CheckOptions{}
	var exitCode bool
	var repo string

	cmd := &cobra.Command{
		Use:   "check <username>",
		Short: "Check the permission of a user for a repository",
		Long:  `Check the permission level (admin, push, maintain, triage, pull, or none) of a specified user for a repository.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			if exitCode {
				cmd.SilenceUsage = true
			}

			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			permission, err := gh.GetRepositoryPermission(ctx, client, repository, username)
			if err != nil {
				return fmt.Errorf("error checking repository permission for user '%s': %w", username, err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderPermission(permission)

			if permission != nil && exitCode {
				cmd.SilenceErrors = true
				return fmt.Errorf("user '%s' has no permissions for the repository", username)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Return an exit code of 1 if the user has no permissions")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
