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
	var noInherit bool
	var owner string
	var roles []string

	cmd := &cobra.Command{
		Use:   "list <team-slug>",
		Short: "List repositories",
		Long:  `List all repositories for the specified team in the organization.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			repos, err := gh.ListTeamRepos(ctx, client, repository, teamSlug, roles, !noInherit)
			if err != nil {
				return fmt.Errorf("failed to list team repositories: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, repos); err != nil {
					return fmt.Errorf("error exporting teams: %w", err)
				}
				return nil
			}

			headers := []string{"NAME", "PERMISSION"}
			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader(headers)

			for _, repo := range repos {
				permission := gh.GetRepositoryPermissions(repo)
				row := []string{
					*repo.FullName,
					permission,
				}
				table.Append(row)
			}
			table.Render()
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&noInherit, "no-inherit", false, "Disable inherited permissions")
	f.StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.PermissionsList, "List of permissions to filter repositories")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
