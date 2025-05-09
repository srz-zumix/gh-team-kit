package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type GetOptions struct {
	Exporter cmdutil.Exporter
}

func init() {
	opts := &GetOptions{}

	var owner string
	var repo string
	var child bool
	var recursive bool
	var getCmd = &cobra.Command{
		Use:   "get [team-slug...]",
		Short: "Gets a team using the team's slug",
		Long:  `Retrieve and display a team using the team's slug.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			teams, err := gh.ListTeamByName(ctx, client, repository, args, child, recursive)
			if err != nil {
				return fmt.Errorf("error retrieving child teams: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, teams); err != nil {
					return fmt.Errorf("error exporting teams: %w", err)
				}
				return nil
			}

			headers := []string{"NAME", "DESCRIPTION"}
			if !child {
				headers = append(headers, "MEMBER_COUNT", "REPOS_COUNT")
			}
			if !child || recursive {
				headers = append(headers, "PARENT_SLUG")
			}

			table := tablewriter.NewWriter(cmd.OutOrStdout())
			table.SetHeader(headers)

			for _, team := range teams {
				data := []string{
					*team.Name,
					*team.Description,
				}
				if !child {
					data = append(data,
						fmt.Sprintf("%d", *team.MembersCount),
						fmt.Sprintf("%d", *team.ReposCount),
					)
				}
				if !child || recursive {
					parentSlug := ""
					if team.Parent != nil && team.Parent.Slug != nil {
						parentSlug = *team.Parent.Slug
					}
					data = append(data, parentSlug)
				}
				table.Append(data)
			}

			table.Render()
			return nil
		},
	}

	f := getCmd.Flags()
	f.StringVarP(&owner, "owner", "", "", "The owner of the team")
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	f.BoolVarP(&child, "child", "c", false, "Retrieve and display the parent team if it exists")
	f.BoolVarP(&recursive, "recursive", "r", false, "Retrieve teams recursively")
	cmdutil.AddFormatFlags(getCmd, &opts.Exporter)

	rootCmd.AddCommand(getCmd)
}
