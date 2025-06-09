package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type GetOptions struct {
	Exporter cmdutil.Exporter
}

func NewGetCmd() *cobra.Command {
	opts := &GetOptions{}

	var owner string
	var repo string
	var child bool
	var recursive bool
	var cmd = &cobra.Command{
		Use:   "get [team-slug...]",
		Short: "Gets a team using the team's slug",
		Long:  `Retrieve and display a team using the team's slug.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			teams, err := gh.ListTeamByName(ctx, client, repository, args, child, recursive)
			if err != nil {
				return fmt.Errorf("error retrieving child teams: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderTeamsDefault(teams)

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	f.BoolVarP(&child, "child", "c", false, "Retrieve and display the parent team if it exists")
	f.BoolVarP(&recursive, "recursive", "r", false, "Retrieve teams recursively")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewGetCmd())
}
