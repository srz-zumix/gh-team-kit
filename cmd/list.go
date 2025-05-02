package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func init() {
	opts := &ListOptions{}
	var repo string

	var teamListCmd = &cobra.Command{
		Use:   "list [owner]",
		Short: "List all teams in the organization",
		Long:  `Retrieve and display a list of all teams in the specified GitHub organization.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			owner := ""
			if len(args) > 0 {
				owner = args[0]
			}
			repository, err := parser.Repository(parser.RepositoryOwner(owner), parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			teams, err := gh.ListTeams(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("error retrieving teams: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, teams); err != nil {
					return fmt.Errorf("error exporting teams: %w", err)
				}
				return nil
			}

			for _, team := range teams {
				fmt.Printf("%s\n", *team.Name)
			}
			return nil
		},
	}

	teamListCmd.Flags().StringVarP(&repo, "repo", "R", "", "Specify a repository to filter teams")
	cmdutil.AddFormatFlags(teamListCmd, &opts.Exporter)

	rootCmd.AddCommand(teamListCmd)
}
