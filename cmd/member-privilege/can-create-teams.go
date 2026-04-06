package memberprivilege

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewCanCreateTeamsCmd creates a command to get or set whether organization members can create teams.
// When --set is specified, the setting is updated and the result is displayed; otherwise the current value is displayed.
func NewCanCreateTeamsCmd() *cobra.Command {
	var owner string
	var exporter cmdutil.Exporter
	var setValue *bool

	cmd := &cobra.Command{
		Use:   "can-create-teams",
		Short: "Get or set whether organization members can create teams",
		Long:  `Get or set whether organization members can create teams. When --set is specified, the setting is updated and the result is displayed; otherwise the current value is displayed.`,
		Args:  cobra.NoArgs,
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

			if setValue != nil {
				_, err = gh.SetOrgMembersCanCreateTeams(ctx, client, repository, *setValue)
				if err != nil {
					return fmt.Errorf("failed to set can-create-teams setting: %w", err)
				}
			}

			org, err := gh.GetOrgMemberPrivileges(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to get can-create-teams setting: %w", err)
			}
			renderer := render.NewRenderer(exporter)
			return renderer.RenderOrgMemberPrivileges(org, []string{"MEMBERS_CAN_CREATE_TEAMS"})
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")
	cmdutil.NilBoolFlag(cmd, &setValue, "set", "", "Set whether members can create teams")
	cmdutil.AddFormatFlags(cmd, &exporter)

	return cmd
}
