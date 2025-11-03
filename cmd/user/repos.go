package user

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type RepoOptions struct {
	Exporter cmdutil.Exporter
}

func NewReposCmd() *cobra.Command {
	opts := &RepoOptions{}
	var archived cmdflags.MutuallyExclusiveBoolFlags
	var fork cmdflags.MutuallyExclusiveBoolFlags
	var mirror cmdflags.MutuallyExclusiveBoolFlags
	var template cmdflags.MutuallyExclusiveBoolFlags
	var nameOnly bool
	var owner string
	var roles []string
	var visibilities []string
	var sources bool

	cmd := &cobra.Command{
		Use:     "repos [username]",
		Short:   "List repositories of a user",
		Long:    `List all repositories owned by the specified user`,
		Aliases: []string{"ls-repo", "repo"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := ""
			if len(args) > 0 {
				username = args[0]
			}

			if sources && (fork.IsEnabled() || mirror.IsEnabled() || archived.IsEnabled()) {
				return fmt.Errorf("the 'sources' option cannot be used with 'fork', 'mirror', or 'archived' options")
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("failed to parse repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			searchOptions := gh.RespositorySearchOptions{
				Visibility: visibilities,
			}
			if archived.IsEnabled() {
				searchOptions.SetArchived(true)
			} else if archived.IsDisabled() {
				searchOptions.SetArchived(false)
			}
			if fork.IsEnabled() {
				searchOptions.SetFork(true)
			} else if fork.IsDisabled() {
				searchOptions.SetFork(false)
			}
			if mirror.IsEnabled() {
				searchOptions.SetMirror(true)
			} else if mirror.IsDisabled() {
				searchOptions.SetMirror(false)
			}
			if template.IsEnabled() {
				searchOptions.SetTemplate(true)
			} else if template.IsDisabled() {
				searchOptions.SetTemplate(false)
			}
			if sources {
				searchOptions.Sources()
			}

			repos, err := gh.ListUserAccessableRepositories(ctx, client, repository, username, roles, &searchOptions)
			if err != nil {
				return fmt.Errorf("failed to list repositories for user '%s': %w", username, err)
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
	f.StringVar(&owner, "owner", "", "Specify the owner of the repository")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.PermissionsList, "List of permissions to filter repositories")
	cmdutil.StringSliceEnumFlag(cmd, &visibilities, "visibility", "v", nil, gh.RepoVisibilityList, "List of visibility to filter repositories")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	f.BoolVar(&sources, "sources", false, "Include only source repositories")
	archived.AddNoPrefixFlag(cmd, "archived", "Include only archived repositories", "Exclude archived repositories")
	fork.AddNoPrefixFlag(cmd, "fork", "Include only forked repositories", "Exclude forked repositories")
	mirror.AddNoPrefixFlag(cmd, "mirror", "Include only mirrored repositories", "Exclude mirrored repositories")
	template.AddFlag(cmd, "is-template", "no-template", "Include only template repositories", "Exclude template repositories")

	return cmd
}
