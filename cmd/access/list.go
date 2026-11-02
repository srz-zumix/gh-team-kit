package access

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// ListOptions contains output settings for the access list command.
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
		Use:     "list <[[host/]owner/]team-slug>",
		Short:   "List repositories accessible to a team",
		Long:    `List repositories accessible to the specified team.`,
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, teamSlug, err := parser.RepositoryWithTeamSlugs(args[0], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing team: %w", err)
			}
			client, err := gh.NewGitHubClientWithRepo(repo)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}
			repos, err := gh.ListTeamRepos(cmd.Context(), client, repo, teamSlug, roles, !noInherit)
			if err != nil {
				return fmt.Errorf("failed to list repositories for team %s: %w", teamSlug, err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				return renderer.RenderNames(repos)
			}
			return renderer.RenderRepository(repos, nil)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only repository names")
	f.BoolVar(&noInherit, "no-inherit", false, "Disable inherited permissions")
	f.StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.PermissionsList, "List of permissions to filter repositories")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
