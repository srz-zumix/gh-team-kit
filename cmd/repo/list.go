package repo

import (
	"context"
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
	var noInherit bool
	var owner string
	var roles []string

	cmd := &cobra.Command{
		Use:     "list <team-slug>",
		Short:   "List repositories",
		Long:    `List all repositories for the specified team in the organization.`,
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(1),
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

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				renderer.RenderNames(repos)
			} else {
				renderer.RenderRepository(repos)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only repository names")
	f.BoolVar(&noInherit, "no-inherit", false, "Disable inherited permissions")
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.PermissionsList, "List of permissions to filter repositories")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
