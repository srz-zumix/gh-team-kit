package member

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type RoleOptions struct {
	Exporter cmdutil.Exporter
}

func NewRoleCmd() *cobra.Command {
	opts := &RoleOptions{}
	var owner string

	cmd := &cobra.Command{
		Use:   "role <team-slug> <username> <role>",
		Short: "Change the role of a user in a team",
		Long:  `Change the role of a specified user in the specified team in the organization.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(3)(cmd, args); err != nil {
				return err
			}
			role := args[2]
			if slices.Contains(gh.TeamMembershipList, role) {
				return nil
			}
			return fmt.Errorf("invalid role '%s', valid roles are: {%s}", role, strings.Join(gh.TeamMembershipList, "|"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			username := args[1]
			role := args[2]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			user, err := gh.UpdateTeamMemberRole(ctx, client, repository, teamSlug, username, role)
			if err != nil {
				return fmt.Errorf("error updating team member role: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, user); err != nil {
					return fmt.Errorf("error exporting user: %w", err)
				}
				return nil
			}

			fmt.Printf("Successfully updated user '%s' role to '%s' in team '%s'.\n", *user.Login, *user.RoleName, teamSlug)
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
