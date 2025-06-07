package repo

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type CheckOptions struct {
	Exporter cmdutil.Exporter
}

func NewCheckCmd() *cobra.Command {
	opts := &CheckOptions{}
	var repo string
	var exitCode bool
	var submodules bool

	cmd := &cobra.Command{
		Use:   "check <team-slug>",
		Short: "Checks whether a team permission for a repository.",
		Long:  `Checks whether a team has admin, push, maintain, triage, pull, or none permission for a repository.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if exitCode {
				cmd.SilenceUsage = true
			}
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			hasPermission := false
			renderer := render.NewRenderer(opts.Exporter)
			if submodules {
				teamRepos, _hasPermission, err := gh.CheckTeamPermissionsWithSubmodules(ctx, client, repository, teamSlug)
				if err != nil {
					return fmt.Errorf("failed to check team permissions for submodules: %w", err)
				}

				hasPermission = _hasPermission
				renderer.RenderPermissions(teamRepos)
			} else {
				teamRepo, _hasPermission, err := gh.CheckTeamPermissions(ctx, client, repository, teamSlug)
				if err != nil {
					return fmt.Errorf("failed to check team permissions: %w", err)
				}

				hasPermission = _hasPermission
				renderer.RenderPermission(teamRepo)
			}

			if !hasPermission && exitCode {
				cmd.SilenceErrors = true
				return fmt.Errorf("no team permissions found for the specified repository")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Exit with a status code based on the result")
	cmd.Flags().BoolVar(&submodules, "submodules", false, "Also check permissions for submodules")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
