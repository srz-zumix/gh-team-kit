package emu

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

// SetOptions holds format flags for the set command.
type SetOptions struct {
	Exporter cmdutil.Exporter
}

// NewSetCmd creates a new cobra.Command for connecting an external group to a team.
func NewSetCmd() *cobra.Command {
	opts := &SetOptions{}
	var owner string
	var fields []string

	cmd := &cobra.Command{
		Use:   "set <group-name> <team-slug>",
		Short: "Connect an external group to a team",
		Long:  "Connect an external group to a team in the organization (Enterprise Managed Users).",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupName := args[0]
			repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[1])
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			group, err := gh.SetExternalGroupForTeam(ctx, client, repository, groupName, teamSlug)
			if err != nil {
				return fmt.Errorf("failed to connect external group '%s' to team '%s': %w", groupName, teamSlug, err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				return renderer.RenderExportedData(group)
			}
			if len(fields) > 0 {
				return renderer.RenderExternalGroup(group, fields)
			} else {
				logger.Info("External group connected to team successfully.", "group-name", groupName, "team-slug", teamSlug)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.ExternalGroupDetailFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
