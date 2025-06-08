package member

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type SetsOptions struct {
	Exporter cmdutil.Exporter
	Sets     gh.SetsOperationFunc
}

// NewSetsCmd creates the `member sets` command
func NewSetsCmd() *cobra.Command {
	opts := &SetsOptions{}
	var details bool
	var nameOnly bool
	var owner string
	var roles []string
	var suspended, noSuspended bool

	cmd := &cobra.Command{
		Use:   "sets <[owner]/team-slug1> <|,&,-,^> <[owner]/team-slug2>",
		Short: "Perform set operations on two teams' members",
		Long:  `Perform set operations on the members of two teams. The operation can be union, intersection, difference, or symmetric difference.`,
		Args:  cobra.ExactArgs(3),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			operation := args[1]
			sets, err := gh.GetSetsOperationFunc(operation)
			if err != nil {
				return err
			}
			opts.Sets = sets
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			team1 := args[0]
			team2 := args[2]

			if suspended || noSuspended {
				details = true
			}
			if suspended && noSuspended {
				return fmt.Errorf("both 'suspended' and 'no-suspended' options cannot be true at the same time")
			}

			repo1, teamSlug1, err := parser.RepositoryFromTeamSlugs(owner, team1)
			if err != nil {
				return fmt.Errorf("error parsing team-slug1 '%s': %w", team1, err)
			}
			repo2, teamSlug2, err := parser.RepositoryFromTeamSlugs(owner, team2)
			if err != nil {
				return fmt.Errorf("error parsing team-slug2 '%s': %w", team1, err)
			}

			ctx := context.Background()
			client1, err := gh.NewGitHubClientWithRepo(repo1)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}
			client2, err := gh.NewGitHubClientWithRepo(repo2)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			// Fetch members for team1 and team2 using the correct teamSlug
			members1, err := gh.ListTeamMembers(ctx, client1, repo1, teamSlug1, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list members of team1 '%s': %w", team1, err)
			}

			members2, err := gh.ListTeamMembers(ctx, client2, repo2, teamSlug2, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list members of team2 '%s': %w", team2, err)
			}

			if details && repo1.Host != repo2.Host {
				// If the repositories are on different hosts, we need to update the user details
				members1, err = gh.UpdateUsers(ctx, client1, members1)
				if err != nil {
					return fmt.Errorf("failed to update users after set operation: %w", err)
				}
				members2, err = gh.UpdateUsers(ctx, client1, members2)
				if err != nil {
					return fmt.Errorf("failed to update users after set operation: %w", err)
				}
			}

			// Perform the set operation using PerformSetOperation
			result := opts.Sets(members1, members2)

			if details {
				if repo1.Host == repo2.Host {
					result, err = gh.UpdateUsers(ctx, client1, result)
					if err != nil {
						return fmt.Errorf("failed to update users after set operation: %w", err)
					}
				}
				if suspended {
					result = gh.CollectSuspendedUsers(result)
				}
				if noSuspended {
					result = gh.ExcludeSuspendedUsers(result)
				}
			}

			// Use the renderer to output the result
			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				renderer.RenderNames(result)
			} else {
				if details {
					renderer.RenderUserDetails(result)
				} else {
					renderer.RenderUsers(result, []string{"USERNAME"})
				}
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	f.BoolVar(&nameOnly, "name-only", false, "Output only member names")
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.BoolVar(&suspended, "suspended", false, "Output only suspended members")
	f.BoolVar(&noSuspended, "no-suspended", false, "Exclude suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
