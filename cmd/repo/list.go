package repo

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var nameOnly bool
	var repo string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List teams for a repository",
		Long:    `List the teams with access to the specified repository and their permissions.`,
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			teams, err := gh.ListRepositoryTeams(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to list teams for repository %s: %w", parser.GetRepositoryFullName(repository), err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				return renderer.RenderNames(teams)
			}
			return renderer.RenderTeamsWithPermission(teams)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only team names")
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format '[HOST/]OWNER/REPO' (defaults to the current repository)")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
