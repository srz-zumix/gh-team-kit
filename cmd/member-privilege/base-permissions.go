package memberprivilege

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/google/go-github/v84/github"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewBasePermissionsCmd creates a command to get or set the default repository permission for an organization.
// When --set is specified, the setting is updated and the result is displayed; otherwise the current value is displayed.
func NewBasePermissionsCmd() *cobra.Command {
	var owner string
	var exporter cmdutil.Exporter
	var setValue string

	cmd := &cobra.Command{
		Use:   "base-permissions",
		Short: "Get or set the default repository permission for organization members",
		Long:  `Get or set the default repository permission (base permissions) for organization members. When --set is specified, the setting is updated and the result is displayed; otherwise the current value is displayed.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()

			if cmd.Flags().Changed("set") {
				_, err = gh.EditOrgMemberPrivileges(ctx, client, repository, &github.Organization{
					DefaultRepoPermission: &setValue,
				})
				if err != nil {
					return fmt.Errorf("failed to set base permissions: %w", err)
				}
			}

			org, err := gh.GetOrgMemberPrivileges(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to get base permissions: %w", err)
			}
			renderer := render.NewRenderer(exporter)
			return renderer.RenderOrgMemberPrivileges(org, []string{"DEFAULT_REPO_PERMISSION"})
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.StringEnumFlag(cmd, &setValue, "set", "", "", gh.OrgDefaultRepoPermissionList, "Set the default repository permission")
	cmdutil.AddFormatFlags(cmd, &exporter)

	return cmd
}
