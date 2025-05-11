package member

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type SetsOptions struct {
	Exporter cmdutil.Exporter
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
		Use:   "sets <team-slug1> <+|*|-> <team-slug2>",
		Short: "Perform set operations on two teams' members",
		Args:  cobra.ExactArgs(3),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			operation := args[1]
			if operation != "+" && operation != "*" && operation != "-" {
				return fmt.Errorf("invalid operation: %s, must be one of +, *, -", operation)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			team1 := args[0]
			operation := args[1]
			team2 := args[2]

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
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			members1, err := gh.ListTeamMembers(ctx, client, repository, team1, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list members of team %s: %w", team1, err)
			}

			members2, err := gh.ListTeamMembers(ctx, client, repository, team2, roles, !nameOnly)
			if err != nil {
				return fmt.Errorf("failed to list members of team %s: %w", team2, err)
			}

			// Perform the set operation using PerformSetOperation
			result, err := gh.PerformSetOperation(members1, members2, operation)
			if err != nil {
				return fmt.Errorf("failed to perform set operation '%s' on teams '%s' and '%s': %w", operation, team1, team2, err)
			}

			if details {
				result, err = gh.UpdateUsers(ctx, client, result)
				if err != nil {
					return fmt.Errorf("failed to update users after set operation: %w", err)
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
			} else if details {
				renderer.RenderUserDetails(result)
			} else {
				renderer.RenderUser(result)
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	f.BoolVarP(&nameOnly, "name-only", "", false, "Output only member names")
	f.StringVarP(&owner, "owner", "", "", "The owner of the team")
	f.BoolVarP(&suspended, "suspended", "", false, "Output only suspended members")
	f.BoolVarP(&noSuspended, "no-suspended", "", false, "Exclude suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
