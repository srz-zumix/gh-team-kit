package access

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// DiffOptions contains output settings for the access diff command.
type DiffOptions struct {
	Exporter cmdutil.Exporter
}

func accessDiffCommandBuilder(left, right string, target string) string {
	return fmt.Sprintf("gh team-kit access diff %s %s %s", left, right, target)
}

func NewDiffCmd() *cobra.Command {
	opts := &DiffOptions{}
	var exitCode bool
	var noInherit bool
	var owner string
	var color string

	cmd := &cobra.Command{
		Use:   "diff <[[host/]owner/]team-slug1> <[[host/]owner/]team-slug2> [repository...]",
		Short: "Compare repositories between two teams",
		Long:  `Compare the repositories associated with two teams and display their permission differences. Use --no-inherit to compare only permissions directly assigned to each team.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if exitCode {
				cmd.SilenceUsage = true
			}
			repo1, teamSlug1, err := parser.RepositoryWithTeamSlugs(args[0], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing first team: %w", err)
			}
			repo2, teamSlug2, err := parser.RepositoryWithTeamSlugs(args[1], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing second team: %w", err)
			}
			client1, client2, err := gh.NewGitHubClientWith2Repos(repo1, repo2)
			if err != nil {
				return fmt.Errorf("error creating GitHub clients: %w", err)
			}
			repos1, err := gh.ListTeamRepos(cmd.Context(), client1, repo1, teamSlug1, nil, !noInherit)
			if err != nil {
				return fmt.Errorf("failed to fetch repositories for team %s: %w", teamSlug1, err)
			}
			repos2, err := gh.ListTeamRepos(cmd.Context(), client2, repo2, teamSlug2, nil, !noInherit)
			if err != nil {
				return fmt.Errorf("failed to fetch repositories for team %s: %w", teamSlug2, err)
			}
			if len(args) > 2 {
				repositories := args[2:]
				repos1 = gh.FilterRepositoriesByNames(repos1, repositories, repo1.Owner)
				repos2 = gh.FilterRepositoriesByNames(repos2, repositories, repo2.Owner)
			}
			differences := gh.CompareRepositories(repos1, repos2)
			renderer := render.NewRenderer(opts.Exporter)
			renderer.SetColor(color)
			commandBuilder := accessDiffCommandBuilder
			if noInherit {
				commandBuilder = func(left, right, target string) string {
					return accessDiffCommandBuilder(left, right, target) + " --no-inherit"
				}
			}
			// Preserve empty output with older versions of go-gh-extension.
			if len(differences) > 0 || renderer.HasExporter() {
				if err := renderer.RenderDiff(differences, teamSlug1, teamSlug2, commandBuilder); err != nil {
					return fmt.Errorf("failed to render team repository differences: %w", err)
				}
			}
			if len(differences) == 0 {
				logger.Info("No differences found.", "left", args[0], "right", args[1])
			}
			if exitCode && len(differences) > 0 {
				cmd.SilenceErrors = true
				return fmt.Errorf("differences found between the teams")
			}
			return nil
		},
	}

	f := cmd.Flags()
	cmdutil.StringEnumFlag(cmd, &color, "color", "", render.ColorFlagAuto, render.ColorFlags, "Use color in diff output")
	f.BoolVar(&exitCode, "exit-code", false, "Return exit code 1 if there are differences")
	f.BoolVar(&noInherit, "no-inherit", false, "Exclude permissions inherited from parent teams")
	f.StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
