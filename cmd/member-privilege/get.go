package memberprivilege

import (
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

// NewGetCmd creates a command to get the member privileges of an organization.
func NewGetCmd() *cobra.Command {
	opts := &GetOptions{}
	var owner string
	var fields []string

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Get member privileges of an organization",
		Long:    `Get the member privileges settings of the specified organization.`,
		Aliases: []string{"view"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			org, err := gh.GetOrgMemberPrivileges(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to get member privileges: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			return renderer.RenderOrgMemberPrivileges(org, fields)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "fields", "", nil, render.OrgMemberPrivilegeFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
