package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type DiffOptions struct {
	Exporter cmdutil.Exporter
}

func commandBuilder(left, right string, target string) string {
	return fmt.Sprintf("gh team-kit repo diff %s %s %s", left, right, target)
}

func NewDiffCmd() *cobra.Command {
	opts := &DiffOptions{}
	var colorFlag string
	var exitCode bool
	var owner string

	cmd := &cobra.Command{
		Use:   "diff <repo1> <repo2> [team-slug...]",
		Short: "Compare team permissions between two repositories",
		Long:  `Compare team permissions between two repositories. The repositories can be specified by their full name (owner/repo) or just the repo name if the owner is provided as a flag.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo1 := args[0]
			repo2 := args[1]

			if exitCode {
				cmd.SilenceUsage = true
			}

			if !strings.Contains(repo1, "/") {
				repo1 = fmt.Sprintf("%s/%s", owner, repo1)
			}
			if !strings.Contains(repo2, "/") {
				repo2 = fmt.Sprintf("%s/%s", owner, repo2)
			}

			repo1Parsed, err := parser.Repository(parser.RepositoryInput(repo1))
			if err != nil {
				return fmt.Errorf("error parsing repository 1: %w", err)
			}

			repo2Parsed, err := parser.Repository(parser.RepositoryInput(repo2))
			if err != nil {
				return fmt.Errorf("error parsing repository 2: %w", err)
			}

			client1, client2, err := gh.NewGitHubClientWith2Repos(repo1Parsed, repo2Parsed)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := context.Background()

			teams1, err := gh.ListRepositoryTeams(ctx, client1, repo1Parsed)
			if err != nil {
				return fmt.Errorf("failed to fetch teams for %s: %w", repo1, err)
			}

			teams2, err := gh.ListRepositoryTeams(ctx, client2, repo2Parsed)
			if err != nil {
				return fmt.Errorf("failed to fetch teams for %s: %w", repo2, err)
			}

			if len(args) > 2 {
				slugs := args[2:]
				teams1 = gh.FilterTeamByNames(teams1, slugs)
				teams2 = gh.FilterTeamByNames(teams2, slugs)
			}

			differences, err := gh.CompareTeamsPermissions(teams1, teams2)
			if err != nil {
				return fmt.Errorf("error comparing team permissions: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			renderer.SetColor(colorFlag)
			renderer.RenderDiff(differences, repo1Parsed, repo2Parsed, commandBuilder)

			if exitCode && len(differences) > 0 {
				cmd.SilenceErrors = true
				return fmt.Errorf("differences found between the repositories")
			}

			return nil
		},
	}

	f := cmd.Flags()
	cmdutil.StringEnumFlag(cmd, &colorFlag, "color", "", render.ColorFlagAuto, render.ColorFlags, "Use color in diff output")
	f.BoolVar(&exitCode, "exit-code", false, "Return exit code 1 if there are differences")
	f.StringVar(&owner, "owner", "", "The owner of the repositories")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
