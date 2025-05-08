package repo

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type CheckOptions struct {
	Exporter cmdutil.Exporter
}

func NewCheckCmd() *cobra.Command {
	opts := &CheckOptions{}
	var repo string
	var exitCode bool

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

			teamRepo, err := gh.CheckTeamPermissions(ctx, client, repository, teamSlug)
			if err != nil {
				return fmt.Errorf("failed to check team permissions: %w", err)
			}

			if opts.Exporter != nil {
				var permissions *map[string]bool
				if teamRepo != nil {
					permissions = &teamRepo.Permissions
				}
				if err := client.Write(opts.Exporter, permissions); err != nil {
					return fmt.Errorf("error exporting team permissions: %w", err)
				}
				return nil
			}

			if teamRepo != nil {
				fmt.Printf("%s\n", gh.GetRepositoryPermissions(teamRepo))
			} else {
				fmt.Printf("none\n")
				if exitCode {
					cmd.SilenceErrors = true
					return fmt.Errorf("no team permissions found for the specified repository")
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Exit with a status code based on the result")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
