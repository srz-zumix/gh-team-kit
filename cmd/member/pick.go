package member

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/google/go-github/v73/github"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type PickOptions struct {
	Exporter cmdutil.Exporter
}

func NewPickCmd() *cobra.Command {
	opts := &PickOptions{}
	var details bool
	var nameOnly bool
	var owner string
	var roles []string
	var suspended, noSuspended bool
	var excludeMembers []string

	cmd := &cobra.Command{
		Use:   "pick <team-slug> [count]",
		Short: "Randomly pick members from a team",
		Long:  `Randomly select a specified number of members from the team. The count parameter specifies how many members to pick. If count is 0 (default), all members are returned. If count is negative, it picks (total members - |count|) members.`,
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			count := 0 // Default value

			if len(args) > 1 {
				countStr := args[1]
				var err error
				count, err = strconv.Atoi(countStr)
				if err != nil {
					return fmt.Errorf("invalid count value '%s': must be a number", countStr)
				}
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

			members, err := gh.ListTeamMembers(ctx, client, repository, teamSlug, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list team members: %w", err)
			}

			if len(members) == 0 {
				return fmt.Errorf("no members found in team '%s'", teamSlug)
			}

			// Apply exclude filter first
			if len(excludeMembers) > 0 {
				var filteredMembers []*github.User
				for _, member := range members {
					excluded := false
					for _, excludeName := range excludeMembers {
						if member.Login != nil && *member.Login == excludeName {
							excluded = true
							break
						}
					}
					if !excluded {
						filteredMembers = append(filteredMembers, member)
					}
				}
				members = filteredMembers

				// Re-check count after excluding members
				if len(members) == 0 {
					return fmt.Errorf("no members found after excluding specified members")
				}
			}

			// Apply filters if details are requested
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

				// Re-check count after filtering
				if len(members) == 0 {
					return fmt.Errorf("no members found after applying filters")
				}
			}

			// Adjust count based on the rules
			if count == 0 {
				count = len(members) // Return all members
			} else if count < 0 {
				count = len(members) + count // Subtract absolute value from total
				if count < 0 {
					return fmt.Errorf("negative count value (%d) results in invalid selection: would need to pick %d members but only %d available", count-len(members), count, len(members))
				}
			}

			if count > len(members) {
				return fmt.Errorf("requested count (%d) is greater than available members (%d)", count, len(members))
			}

			// Randomly pick members
			// Shuffle and pick the specified count
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			r.Shuffle(len(members), func(i, j int) {
				members[i], members[j] = members[j], members[i]
			})
			pickedMembers := members[:count]

			renderer := render.NewRenderer(opts.Exporter)

			if nameOnly {
				renderer.RenderNames(pickedMembers)
				return nil
			} else {
				if details {
					renderer.RenderUserDetails(pickedMembers)
				} else {
					renderer.RenderUserWithRole(pickedMembers)
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
	f.StringSliceVarP(&excludeMembers, "exclude", "e", nil, "Exclude specified members from pick selection")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
