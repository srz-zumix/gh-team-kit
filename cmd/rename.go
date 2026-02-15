package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type RenameOptions struct {
	Exporter cmdutil.Exporter
}

func NewRenameCmd() *cobra.Command {
	opts := &RenameOptions{}
	var owner string

	cmd := &cobra.Command{
		Use:     "rename <team-slug> <new-name>",
		Short:   "Rename an existing team",
		Long:    `Rename an existing team in the specified organization to a new name.`,
		Aliases: []string{"rn"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			newName := args[1]

			repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[0])
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			team, err := gh.RenameTeam(ctx, client, repository, teamSlug, newName)
			if err != nil {
				return fmt.Errorf("failed to rename team: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				renderer.RenderExportedData(team)
				return nil
			}

			logger.Info("Team renamed successfully.", "before", teamSlug, "after", newName)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization owner")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewRenameCmd())
}
