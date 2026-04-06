package emu

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// TeamsOptions holds format flags for the teams command.
type TeamsOptions struct {
	Exporter cmdutil.Exporter
}

// NewTeamsCmd creates a new cobra.Command for listing teams connected to an external group.
func NewTeamsCmd() *cobra.Command {
	opts := &TeamsOptions{}
	var owner string
	var fields []string

	cmd := &cobra.Command{
		Use:   "teams <group-name>",
		Short: "List teams connected to an external group",
		Long:  "List the teams connected to an external group, with detailed team info fetched from the organization (Enterprise Managed Users).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			groupName := args[0]

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			teams, err := gh.GetExternalGroupTeams(ctx, client, repository, groupName)
			if err != nil {
				return fmt.Errorf("failed to get teams for external group '%s': %w", groupName, err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			return renderer.RenderExternalGroupTeamDetails(teams, fields)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.ExternalGroupTeamDetailFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
