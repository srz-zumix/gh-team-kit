package member

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type AddOptions struct {
	Exporter cmdutil.Exporter
}

func NewAddCmd() *cobra.Command {
	opts := &AddOptions{}
	var allowNonOrganizationMember bool
	var owner string
	var role string

	cmd := &cobra.Command{
		Use:   "add <team-slug> <username>",
		Short: "Add a member to a team",
		Long:  `Add a specified user to the specified team in the organization.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			username := args[1]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			membership, err := gh.AddTeamMember(ctx, client, repository, teamSlug, username, role, allowNonOrganizationMember)
			if err != nil {
				return fmt.Errorf("failed to add member to team: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, membership); err != nil {
					return fmt.Errorf("error exporting membership: %w", err)
				}
				return nil
			}

			fmt.Printf("Successfully added user '%s' to team '%s' with role '%s'.\n", username, teamSlug, *membership.Role)
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&allowNonOrganizationMember, "allow-non-organization-member", "", false, "Allow adding non-organization member to the team")
	f.StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmdutil.StringEnumFlag(cmd, &role, "role", "", "member", gh.TeamMembershipList, "Role to assign to the user (default: member)").NoOptDefVal = "member"
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
