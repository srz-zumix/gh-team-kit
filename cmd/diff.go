package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type DiffOptions struct {
	Exporter cmdutil.Exporter
}

var colorFlag string

func NewDiffCmd() *cobra.Command {
	opts := &DiffOptions{}
	var exitCode bool
	var owner string

	cmd := &cobra.Command{
		Use:   "diff <team-slug1> <team-slug2> [repository...]",
		Short: "Compare repositories between two teams",
		Long:  `The diff command compares the repositories associated with two teams and displays the differences.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug1 := args[0]
			teamSlug2 := args[1]

			if exitCode {
				cmd.SilenceUsage = true
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository owner: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
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

			if len(args) > 2 {
				repositories := args[2:]
				repos1 = gh.FilterRepositoriesByNames(repos1, repositories, repository.Owner)
				repos2 = gh.FilterRepositoriesByNames(repos2, repositories, repository.Owner)
			}

			differences := gh.CompareRepositories(repos1, repos2)

			renderer := render.NewRenderer(opts.Exporter)
			renderer.SetColor(colorFlag)
			renderer.RenderDiff(differences, teamSlug1, teamSlug2)

			if exitCode && len(differences) > 0 {
				cmd.SilenceErrors = true
				return fmt.Errorf("differences found between the teams")
			}

			return nil
		},
	}

	f := cmd.Flags()
	cmdutil.StringEnumFlag(cmd, &colorFlag, "color", "", "auto", []string{"always", "never", "auto"}, "Use color in diff output")
	cmd.Flags().BoolVar(&exitCode, "exit-code", false, "Return exit code 1 if there are differences")
	f.StringVarP(&owner, "owner", "", "", "Specify the organization name")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewDiffCmd())
}
