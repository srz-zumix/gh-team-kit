package member

import (
	"context"
	"fmt"
	"slices"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var details bool
	var nameOnly bool
	var owner string
	var roles []string
	var suspended bool

	cmd := &cobra.Command{
		Use:   "list <team-slug>",
		Short: "List members of a team",
		Long:  `List all members of the specified team in the organization.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			if suspended {
				details = true
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

			if details {
				members, err = gh.UpdateUsers(ctx, client, members)
				if err != nil {
					return fmt.Errorf("failed to update users: %w", err)
				}
				if suspended {
					members = slices.DeleteFunc(members, func(member *github.User) bool {
						return member.SuspendedAt == nil
					})
				}
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, members); err != nil {
					return fmt.Errorf("error exporting team members: %w", err)
				}
				return nil
			}

			if nameOnly {
				for _, member := range members {
					fmt.Println(*member.Login)
				}
				return nil
			}

			headers := []string{"USERNAME", "ROLE"}
			if details {
				headers = append(headers, "EMAIL", "SUSPENDED")
			}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader(headers)

			for _, member := range members {
				row := []string{
					*member.Login,
					*member.RoleName,
				}
				if details {
					if member.Email != nil {
						row = append(row, *member.Email)
					} else {
						row = append(row, "")
					}
					if member.SuspendedAt != nil {
						row = append(row, "Yes")
					} else {
						row = append(row, "No")
					}
				}
				table.Append(row)
			}
			table.Render()
			return nil
		},
	}

	cmd.Flags().BoolVarP(&details, "details", "", false, "Include detailed information about members")
	cmd.Flags().BoolVarP(&nameOnly, "name-only", "", false, "Output only member names")
	cmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmd.Flags().BoolVarP(&suspended, "suspended", "", false, "Output only suspended members")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "", nil, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
