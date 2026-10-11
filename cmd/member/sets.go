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
	var suspended cmdflags.MutuallyExclusiveBoolFlags

	cmd := &cobra.Command{
		Use:   "sets <[owner]/team-slug1|@any|@all> <|,&,-,^> <[owner]/team-slug2|@any|@all>",
		Short: "Perform set operations on two teams' members",
		Long: `Perform set operations on the members of two teams. The operation can be union, intersection, difference, or symmetric difference.

Special team slugs:
  @any  - All members who belong to any team in the organization (union of all teams)
  @all  - All members of the organization`,
		Args: cobra.ExactArgs(3),
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

			fetchDetails := details
			if suspended.IsSet() {
				details = true
			}
			updateUsers := gh.UpdateUsersForSuspension
			if fetchDetails {
				updateUsers = gh.UpdateUsers
			}

			repo1, teamSlug1, err := parser.RepositoryWithTeamSlugs(team1, parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing team-slug1 '%s': %w", team1, err)
			}
			repo2, teamSlug2, err := parser.RepositoryWithTeamSlugs(team2, parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing team-slug2 '%s': %w", team2, err)
			}

			client1, err := gh.NewGitHubClientWithRepo(repo1)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}
			client2, err := gh.NewGitHubClientWithRepo(repo2)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			ctx := cmd.Context()
			// Fetch members for team1 and team2 using the correct teamSlug
			members1, err := gh.ListMembersByTeamSpec(ctx, client1, repo1, teamSlug1, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list members of team1 '%s': %w", team1, err)
			}

			members2, err := gh.ListMembersByTeamSpec(ctx, client2, repo2, teamSlug2, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list members of team2 '%s': %w", team2, err)
			}

			if details && repo1.Host != repo2.Host {
				// If the repositories are on different hosts, we need to update the user details
				members1, err = updateUsers(ctx, client1, members1)
				if err != nil {
					return fmt.Errorf("failed to update users after set operation: %w", err)
				}
				members2, err = updateUsers(ctx, client2, members2)
				if err != nil {
					return fmt.Errorf("failed to update users after set operation: %w", err)
				}
				if suspended.IsSet() && !fetchDetails && !nameOnly {
					// Fetch full details with each host's client for the members kept by the suspension filter
					if _, err = gh.UpdateUsers(ctx, client1, filterSuspendedUsers(suspended, members1)); err != nil {
						return fmt.Errorf("failed to update user details in team1: %w", err)
					}
					if _, err = gh.UpdateUsers(ctx, client2, filterSuspendedUsers(suspended, members2)); err != nil {
						return fmt.Errorf("failed to update user details in team2: %w", err)
					}
				}
			}

			// Perform the set operation using PerformSetOperation
			result := opts.Sets(members1, members2)

			if details {
				if repo1.Host == repo2.Host {
					result, err = updateUsers(ctx, client1, result)
					if err != nil {
						return fmt.Errorf("failed to update users after set operation: %w", err)
					}
				}
				result = filterSuspendedUsers(suspended, result)
				if suspended.IsSet() && !fetchDetails && !nameOnly && repo1.Host == repo2.Host {
					result, err = gh.UpdateUsers(ctx, client1, result)
					if err != nil {
						return fmt.Errorf("failed to update user details after set operation: %w", err)
					}
				}
			}

			// Use the renderer to output the result
			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				return renderer.RenderNames(result)
			}
			if details {
				return renderer.RenderUserDetails(result)
			} else {
				return renderer.RenderUsers(result, []string{"USERNAME"})
			}
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	f.BoolVar(&nameOnly, "name-only", false, "Output only member names")
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")
	suspended.AddNoPrefixFlag(cmd, "suspended", "Output only suspended members", "Exclude suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

// filterSuspendedUsers applies the suspension filter selected by the flag.
func filterSuspendedUsers(suspended cmdflags.MutuallyExclusiveBoolFlags, users []*gh.GitHubUser) []*gh.GitHubUser {
	if suspended.IsEnabled() {
		return gh.CollectSuspendedUsers(users)
	}
	if suspended.IsDisabled() {
		return gh.ExcludeSuspendedUsers(users)
	}
	return users
}
