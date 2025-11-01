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

type AddOptions struct {
	Exporter cmdutil.Exporter
}

func NewAddCmd() *cobra.Command {
	opts := &AddOptions{}
	var allowNonOrganizationMember bool
	var owner string
	var role string

	cmd := &cobra.Command{
		Use:   "add <team-slug> <username...>",
		Short: "Add a member to a team",
		Long:  `Add a specified user to the specified team in the organization.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			usernames := args[1:]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			memberships, err := gh.AddTeamMembers(ctx, client, repository, teamSlug, usernames, role, allowNonOrganizationMember)
			if err != nil {
				return fmt.Errorf("failed to add member to team: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				if len(memberships) == 1 {
					renderer.RenderExportedData(memberships[0])
					return nil
				}
				renderer.RenderExportedData(memberships)
				return nil
			}
			for _, membership := range memberships {
				username := membership.User.GetLogin()
				fmt.Printf("Successfully added user '%s' to team '%s' with role '%s'.\n", username, teamSlug, *membership.Role)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&allowNonOrganizationMember, "allow-non-organization-member", false, "Allow adding non-organization member to the team")
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.StringEnumFlag(cmd, &role, "role", "r", "member", gh.TeamMembershipList, "Role to assign to the user (default: member)").NoOptDefVal = "member"
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
