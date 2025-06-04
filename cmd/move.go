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

type MoveOptions struct {
	Exporter cmdutil.Exporter
}

func NewMoveCmd() *cobra.Command {
	opts := &MoveOptions{}
	var owner string

	cmd := &cobra.Command{
		Use:     "move <team-slug> [new-parent-slug]",
		Short:   "Change the parent of an existing team",
		Long:    `Change the parent of an existing team in the specified organization to a new parent team. if no new parent is specified, the team will be moved to the root level.`,
		Aliases: []string{"mv"},
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			newParent := ""
			if len(args) > 1 {
				newParent = args[1]
			}

			ctx := context.Background()

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			team, err := gh.UpdateTeam(ctx, client, repository, teamSlug, nil, nil, nil, nil, &newParent)
			if err != nil {
				return fmt.Errorf("failed to update team parent: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				renderer.RenderExportedData(team)
				return nil
			}

			if newParent == "" {
				fmt.Printf("Team '%s' moved to root level successfully.\n", teamSlug)
			} else {
				fmt.Printf("Team '%s' parent changed to '%s' successfully.\n", teamSlug, newParent)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization owner")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewMoveCmd())
}
