package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var nameOnly bool
	var repo string

	var cmd = &cobra.Command{
		Use:   "list [owner]",
		Short: "List all teams in the organization",
		Long:  `Retrieve and display a list of all teams in the specified organization. You can optionally filter the results by repository.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := ""
			if len(args) > 0 {
				owner = args[0]
			}
			repository, err := parser.Repository(parser.RepositoryOwner(owner), parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			teams, err := gh.ListTeams(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("error retrieving teams: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				renderer.RenderNames(teams)
			} else {
				renderer.RenderTeamsWithPermission(teams)
			}
			return nil
		},
	}

	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	cmd.Flags().BoolVarP(&nameOnly, "name-only", "", false, "Output only team names")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "Specify a repository to filter teams")

	return cmd
}

func init() {
	rootCmd.AddCommand(NewListCmd())
}
