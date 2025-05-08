package member

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

func NewAddCmd() *cobra.Command {
	var allowNonOrganizationMember bool
	var owner string

	cmd := &cobra.Command{
		Use:   "add <team-slug> <username> [role]",
		Short: "Add a member to a team",
		Long:  `Add a specified user to the specified team in the organization. Optionally specify the role (default: member).`,
		Args: func(cmd *cobra.Command, args []string) error {
			err := cobra.RangeArgs(2, 3)(cmd, args)
			if err != nil {
				return err
			}
			if len(args) == 3 {
				role := args[2]
				for _, valid := range gh.TeamMembershipList {
					if role == valid {
						return nil
					}
				}
				return fmt.Errorf("invalid role '%s', valid roles are: {%s}", role, strings.Join(gh.TeamMembershipList, "|"))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			username := args[1]
			role := "member"
			if len(args) == 3 {
				role = args[2]
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

			if err := gh.AddTeamMember(ctx, client, repository, teamSlug, username, role, allowNonOrganizationMember); err != nil {
				return fmt.Errorf("failed to add member to team: %w", err)
			}

			fmt.Printf("Successfully added user '%s' to team '%s' with role '%s'.\n", username, teamSlug, role)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&allowNonOrganizationMember, "allow-non-organization-member", "", false, "Allow adding non-organization member to the team")
	cmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the team")
	return cmd
}
