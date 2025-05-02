package repo

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

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories",
		Long:  `List all repositories for the specified team in the organization.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			teamSlug := args[0]
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				fmt.Printf("Error parsing repository: %v\n", err)
				return
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				fmt.Printf("Error creating GitHub client: %v\n", err)
				return
			}

			repos, err := gh.ListTeamRepos(ctx, client, repository, teamSlug, roles)
			if err != nil {
				fmt.Printf("Failed to list team repositories: %v\n", err)
				return
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, repos); err != nil {
					fmt.Printf("Error exporting teams: %v\n", err)
					return
				}
				return
			}

			headers := []string{"NAME", "PERMISSION"}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader(headers)

			for _, repo := range repos {
				row := []string{
					*repo.FullName,
					gh.GetRepositoryPermissions(repo),
				}
				table.Append(row)
			}
			table.Render()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "", []string{}, gh.TeamPermissionsList, "List of roles to filter repositories")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
