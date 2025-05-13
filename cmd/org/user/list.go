package user

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var nameOnly bool
	var owner string
	var details, suspended, noSuspended bool

	cmd := &cobra.Command{
		Use:   "list [org-role-name]",
		Short: "List users assigned to a organization role",
		Long:  `Retrieve and display a list of all users assigned to a specific role in the organization.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			role := ""
			if len(args) > 0 {
				role = args[0]
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

			users, err := gh.ListUsersAssignedToOrgRole(ctx, client, repository, role)
			if err != nil {
				return fmt.Errorf("failed to list users assigned to role '%s': %w", role, err)
			}

			if details {
				users, err = gh.UpdateUsers(ctx, client, users)
				if err != nil {
					return fmt.Errorf("failed to update user details: %w", err)
				}
				if suspended {
					users = gh.CollectSuspendedUsers(users)
				}
				if noSuspended {
					users = gh.ExcludeSuspendedUsers(users)
				}
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				renderer.RenderNames(users)
			} else {
				headers := []string{"USERNAME", "ROLE", "TEAM"}
				if details {
					headers = append(headers, "EMAIL", "SUSPENDED")
				}
				renderer.RenderUsers(users, headers)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	f.BoolVarP(&nameOnly, "name-only", "", false, "Output only user names")
	f.StringVarP(&owner, "owner", "", "", "Specify the organization name")
	f.BoolVarP(&suspended, "suspended", "", false, "Output only suspended members")
	f.BoolVarP(&noSuspended, "no-suspended", "", false, "Exclude suspended members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
