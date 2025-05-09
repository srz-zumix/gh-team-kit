package user

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
	var owner string
	var role string

	cmd := &cobra.Command{
		Use:   "add <username>",
		Short: "Add a user to the organization",
		Long:  `Add a specified user to the organization using the role and username.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("failed to parse owner: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			user, err := gh.AddOrgMember(ctx, client, repository, username, role)
			if err != nil {
				return fmt.Errorf("failed to set organization membership: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, user); err != nil {
					return fmt.Errorf("error exporting user: %w", err)
				}
				return nil
			}
			fmt.Printf("Successfully added user '%s' to the organization with role '%s'.\n", *user.Login, *user.RoleName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "o", "", "Owner of the organization")
	cmdutil.StringEnumFlag(cmd, &role, "role", "", "member", gh.OrgMembershipList, "Role to assign to the user (default: member)").NoOptDefVal = "member"
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
