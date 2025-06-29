package user

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
	var details bool
	var nameOnly bool
	var roles []string
	var suspended, noSuspended bool

	cmd := &cobra.Command{
		Use:   "list [owner]",
		Short: "List organization members",
		Long:  `List all members of the specified organization with optional role filtering`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner string
			if len(args) > 0 {
				owner = args[0]
			}

			if suspended || noSuspended {
				details = true
			}
			if suspended && noSuspended {
				return fmt.Errorf("both 'suspended' and 'no-suspended' options cannot be true at the same time")
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
				if suspended {
					members = gh.CollectSuspendedUsers(members)
				}
				if noSuspended {
					members = gh.ExcludeSuspendedUsers(members)
				}
			}

			if nameOnly {
				renderer.RenderNames(members)
			} else {
				if details {
					renderer.RenderUserDetails(members)
				} else {
					renderer.RenderUserWithRole(members)
				}
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	f.BoolVar(&nameOnly, "name-only", false, "Output only member names")
	f.BoolVar(&suspended, "suspended", false, "Output only suspended members")
	f.BoolVar(&noSuspended, "no-suspended", false, "Exclude suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.OrgMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
