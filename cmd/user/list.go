package user

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var details bool
	var nameOnly bool
	var roles []string
	var suspended cmdflags.MutuallyExclusiveBoolFlags

	cmd := &cobra.Command{
		Use:     "list [owner]",
		Short:   "List organization members",
		Long:    `List all members of the specified organization with optional role filtering`,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner string
			if len(args) > 0 {
				owner = args[0]
			}

			if suspended.IsSet() {
				details = true
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			members, err := gh.ListOrgMembers(ctx, client, repository, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list organization members: %w", err)
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
	suspended.AddNoPrefixFlag(cmd, "suspended", "Output only suspended members", "Exclude suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.OrgMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
