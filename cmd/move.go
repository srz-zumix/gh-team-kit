package cmd

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
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
			newParent := ""
			if len(args) > 1 {
				newParent = args[1]
			}

			repository, teamSlug, err := parser.RepositoryWithTeamSlugs(args[0], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			team, err := gh.UpdateTeam(ctx, client, repository, teamSlug, nil, nil, nil, nil, &newParent)
			if err != nil {
				return fmt.Errorf("failed to update team parent: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				return renderer.RenderExportedData(team)
			}

			if newParent == "" {
				logger.Info("Team moved to root level successfully.", "team-slug", teamSlug)
			} else {
				logger.Info("Team parent changed successfully.", "team-slug", teamSlug, "new-parent", newParent)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewMoveCmd())
}
