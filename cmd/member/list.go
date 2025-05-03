package member

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
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
	var owner string
	var roles []string
	var nameOnly bool

	cmd := &cobra.Command{
		Use:   "list <team-slug>",
		Short: "List members of a team",
		Long:  `List all members of the specified team in the organization.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			if nameOnly {
				roles = []string{}
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

			members, err := gh.ListTeamMembers(ctx, client, repository, teamSlug, roles)
			if err != nil {
				return fmt.Errorf("failed to list team members: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, members); err != nil {
					return fmt.Errorf("error exporting team members: %w", err)
				}
				return nil
			}

			if nameOnly || len(roles) == 0 {
				for _, member := range members {
					fmt.Fprintln(cmd.OutOrStdout(), *member.Login)
				}
				return nil
			}

			headers := []string{"USERNAME", "ROLE"}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader(headers)

			for _, member := range members {
				row := []string{
					*member.Login,
					*member.RoleName,
				}
				table.Append(row)
			}
			table.Render()
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmd.Flags().BoolVarP(&nameOnly, "name-only", "", false, "Output only member names")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "", gh.TeamMembershipList, gh.TeamMembershipList, "List of roles to filter members")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
