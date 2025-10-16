package member

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/cli/cli/v2/pkg/cmdutil"
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

	cmd := &cobra.Command{
		Use:   "pick <team-slug> <count>",
		Short: "Randomly pick members from a team",
		Long:  `Randomly select a specified number of members from the team. The count parameter specifies how many members to pick.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			countStr := args[1]

			count, err := strconv.Atoi(countStr)
			if err != nil {
				return fmt.Errorf("invalid count value '%s': must be a number", countStr)
			}
			if count <= 0 {
				return fmt.Errorf("count must be greater than 0")
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

			if count > len(members) {
				return fmt.Errorf("requested count (%d) is greater than available members (%d)", count, len(members))
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
				if count > len(members) {
					return fmt.Errorf("requested count (%d) is greater than available members after filtering (%d)", count, len(members))
				}
			}

			// Randomly pick members
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
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
