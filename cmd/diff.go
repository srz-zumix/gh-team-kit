package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type DiffOptions struct {
	Exporter cmdutil.Exporter
	Color    bool
}

var colorFlag string

func NewDiffCmd() *cobra.Command {
	opts := &DiffOptions{}
	var owner string

	cmd := &cobra.Command{
		Use:   "diff <team-slug1> <team-slug2> [repository...]",
		Short: "Compare repositories between two teams",
		Long:  `The diff command compares the repositories associated with two teams and displays the differences.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug1 := args[0]
			teamSlug2 := args[1]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository owner: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if colorFlag == "always" || (colorFlag == "auto" && client.IO.ColorEnabled()) {
				opts.Color = true
			} else {
				opts.Color = false
			}

			ctx := context.Background()

			repos1, err := gh.ListTeamRepos(ctx, client, repository, teamSlug1, nil, true)
			if err != nil {
				return fmt.Errorf("error fetching repositories for team %s: %w", teamSlug1, err)
			}

			repos2, err := gh.ListTeamRepos(ctx, client, repository, teamSlug2, nil, true)
			if err != nil {
				return fmt.Errorf("error fetching repositories for team %s: %w", teamSlug2, err)
			}

			// Parse repositories from arguments
			var repositories []string
			if len(args) > 2 {
				repositories = args[2:]
			}

			if len(repositories) > 0 {
				repos1 = gh.FilterRepositoriesByNames(repos1, repositories, repository.Owner)
				repos2 = gh.FilterRepositoriesByNames(repos2, repositories, repository.Owner)
			}

			differences := gh.CompareRepositories(repos1, repos2)

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, differences); err != nil {
					return fmt.Errorf("error exporting differences: %w", err)
				}
				return nil
			}

			if opts.Color {
				fmt.Printf("%s", colorizeDiff(differences.GetDiffLines(teamSlug1, teamSlug2)))
			} else {
				fmt.Printf("%s", differences.GetDiffLines(teamSlug1, teamSlug2))
			}

			return nil
		},
	}

	f := cmd.Flags()
	cmdutil.StringEnumFlag(cmd, &colorFlag, "color", "", "auto", []string{"always", "never", "auto"}, "Use color in diff output")
	f.StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func colorizeDiff(diff string) string {
	var result string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+ ") {
			result += color.GreenString(line) + "\n"
		} else if strings.HasPrefix(line, "- ") {
			result += color.RedString(line) + "\n"
		} else {
			result += line + "\n"
		}
	}
	return result
}

func init() {
	rootCmd.AddCommand(NewDiffCmd())
}
