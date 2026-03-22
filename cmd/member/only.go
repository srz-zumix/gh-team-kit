package member

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type OnlyOptions struct {
	Exporter cmdutil.Exporter
}

// NewOnlyCmd creates the `member only` command
func NewOnlyCmd() *cobra.Command {
	opts := &OnlyOptions{}
	var details bool
	var nameOnly bool
	var owner string
	var roles []string
	var suspended cmdflags.MutuallyExclusiveBoolFlags

	cmd := &cobra.Command{
		Use:   "only <team-slug>",
		Short: "List members who belong only to the specified team",
		Long:  `List members who belong exclusively to the specified team and are not members of any other team in the organization.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if suspended.IsSet() {
				details = true
			}

			repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[0])
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			members, err := gh.ListOnlyTeamMembers(ctx, client, repository, teamSlug, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list exclusive members: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)

			if details {
				members, err = gh.UpdateUsers(ctx, client, members)
				if err != nil {
					return fmt.Errorf("failed to update users: %w", err)
				}
				if suspended.IsEnabled() {
					members = gh.CollectSuspendedUsers(members)
				}
				if suspended.IsDisabled() {
					members = gh.ExcludeSuspendedUsers(members)
				}
			}

			if nameOnly {
				return renderer.RenderNames(members)
			}
			if details {
				return renderer.RenderUserDetails(members)
			} else {
				return renderer.RenderUserWithRole(members)
			}
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	f.BoolVar(&nameOnly, "name-only", false, "Output only member names")
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	suspended.AddNoPrefixFlag(cmd, "suspended", "Output only suspended members", "Exclude suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
