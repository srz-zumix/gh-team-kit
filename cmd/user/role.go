package user

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

// RoleOptions holds options for the role command
type RoleOptions struct {
	Exporter cmdutil.Exporter
}

// NewRoleCmd creates a new role command for users
func NewRoleCmd() *cobra.Command {
	opts := &RoleOptions{}
	var owner string

	cmd := &cobra.Command{
		Use:   "role <username> <role>",
		Short: "Change the role of a user in an organization",
		Long:  `Change the role of a specified user in the specified organization.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				return err
			}
			role := args[1]
			if slices.Contains(gh.OrgMembershipList, role) {
				return nil
			}
			return fmt.Errorf("invalid role '%s', valid roles are: {%s}", role, strings.Join(gh.OrgMembershipList, "|"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
			role := args[1]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			user, err := gh.UpdateOrgMemberRole(ctx, client, repository, username, role)
			if err != nil {
				return fmt.Errorf("error updating user role: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, user); err != nil {
					return fmt.Errorf("error exporting user: %w", err)
				}
				return nil
			}

			fmt.Printf("Successfully updated user '%s' role to '%s' in the organization.\n", *user.Login, *user.RoleName)
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the organization")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
