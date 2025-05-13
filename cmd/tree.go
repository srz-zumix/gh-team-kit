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

type TreeOptions struct {
	Exporter cmdutil.Exporter
}

func NewTreeCmd() *cobra.Command {
	opts := &TreeOptions{}
	var owner string
	var recursive bool

	var cmd = &cobra.Command{
		Use:   "tree [team-slug]",
		Short: "Displays a team hierarchy in a tree structure",
		Long:  `Displays a team hierarchy in a tree structure based on the team's slug.`,
		Args:  cobra.MaximumNArgs(1),
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

			var team gh.Team
			if len(args) > 0 {
				teamSlug := args[0]
				team, err = gh.TeamByName(ctx, client, repository, teamSlug, false, recursive)
				if err != nil {
					return fmt.Errorf("error retrieving teams: %w", err)
				}
			} else {
				team, err = gh.TeamByOwner(ctx, client, repository, recursive)
				if err != nil {
					return fmt.Errorf("error retrieving teams: %w", err)
				}
			}

			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderTeamTree(repository.Owner, team)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "", "", "Specify the organization name")
	f.BoolVarP(&recursive, "recursive", "r", false, "Retrieve teams recursively")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewTreeCmd())
}
