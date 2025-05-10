package user

import (
	"context"
	"fmt"
	"slices"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/google/go-github/v71/github"
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
	var details bool
	var nameOnly bool
	var roles []string
	var suspended bool

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

			if suspended {
				details = true
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
			if nameOnly {
				renderer.RenderNames(members)
				return nil
			}

			if details {
				members, err = gh.UpdateUsers(ctx, client, members)
				if err != nil {
					return fmt.Errorf("failed to update users: %w", err)
				}
				if suspended {
					members = slices.DeleteFunc(members, func(member *github.User) bool {
						return member.SuspendedAt == nil
					})
				}
			}

			if details {
				renderer.RenderUserDetails(members)
			} else {
				renderer.RenderUser(members)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&details, "details", "", false, "Include detailed information about members")
	cmd.Flags().BoolVarP(&nameOnly, "name-only", "", false, "Output only member names")
	cmd.Flags().BoolVarP(&suspended, "suspended", "", false, "Output only suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.OrgMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
