package cmd

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var nameOnly bool
	var repo string

	var cmd = &cobra.Command{
		Use:     "list [owner]",
		Short:   "List all teams in the organization",
		Long:    `Retrieve and display a list of all teams in the specified organization. You can optionally filter the results by repository.`,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := ""
			if len(args) > 0 {
				owner = args[0]
			}
			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner), parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			teams, err := gh.ListTeams(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("error retrieving teams: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				return renderer.RenderNames(teams)
			}
			return renderer.RenderTeamsWithPermission(teams)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only team names")
	f.StringVarP(&repo, "repo", "R", "", "Specify a repository to filter teams")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewListCmd())
}
