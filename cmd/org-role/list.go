package orgrole

import (
	"context"
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
	var fields []string
	var sources []string

	cmd := &cobra.Command{
		Use:     "list [owner]",
		Short:   "List roles in the organization",
		Long:    `List all roles available in the organization.`,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner string
			if len(args) > 0 {
				owner = args[0]
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			roles, err := gh.ListOrgRolesBySource(ctx, client, repository, sources)
			if err != nil {
				return fmt.Errorf("failed to list roles for owner '%s': %w", owner, err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				return renderer.RenderNames(roles)
			} else {
				return renderer.RenderCustomOrgRoles(roles, fields)
			}
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only role names")
	cmdutil.StringSliceEnumFlag(cmd, &sources, "source", "", nil, gh.OrgCustomRoleSourceList, "Filter by role source")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.CustomOrgRoleFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
