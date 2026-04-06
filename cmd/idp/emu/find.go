package emu

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// FindOptions holds format flags for the find command.
type FindOptions struct {
	Exporter cmdutil.Exporter
}

// NewFindCmd creates a new cobra.Command for finding the external group connected to a team.
func NewFindCmd() *cobra.Command {
	opts := &FindOptions{}
	var owner string
	var fields []string

	cmd := &cobra.Command{
		Use:   "find <team-slug>",
		Short: "Find the external group connected to a team",
		Long:  "Find the external group connected to a team in the organization (Enterprise Managed Users). Exits with no output if no external group is connected.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, teamSlug, err := parser.RepositoryWithTeamSlugs(args[0], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			group, err := gh.FindExternalGroupByTeamSlug(ctx, client, repository, teamSlug)
			if err != nil {
				return fmt.Errorf("failed to find external group for team '%s': %w", teamSlug, err)
			}
			if group == nil {
				logger.Info("No external group is connected to the team.", "team-slug", teamSlug)
				return nil
			}

			renderer := render.NewRenderer(opts.Exporter)
			return renderer.RenderExternalGroup(group, fields)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.ExternalGroupDetailFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
