package user

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type RepoOptions struct {
	Exporter cmdutil.Exporter
}

func NewRepoCmd() *cobra.Command {
	opts := &RepoOptions{}
	var archived, noArchived bool
	var fork, noFork bool
	var mirror, noMirror bool
	var template, noTemplate bool
	var nameOnly bool
	var owner string
	var roles []string
	var visibilities []string
	var sources bool

	cmd := &cobra.Command{
		Use:   "repo [username]",
		Short: "List repositories of a user",
		Long:  `List all repositories owned by the specified user`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := ""
			if len(args) > 0 {
				username = args[0]
			}

			if archived && noArchived {
				return fmt.Errorf("both 'archived' and 'no-archived' options cannot be true at the same time")
			}

			if fork && noFork {
				return fmt.Errorf("both 'fork' and 'no-fork' options cannot be true at the same time")
			}

			if mirror && noMirror {
				return fmt.Errorf("both 'mirror' and 'no-mirror' options cannot be true at the same time")
			}

			if template && noTemplate {
				return fmt.Errorf("both 'template' and 'no-template' options cannot be true at the same time")
			}

			if sources && (fork || mirror || archived) {
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
			if archived {
				searchOptions.SetArchived(true)
			} else if noArchived {
				searchOptions.SetArchived(false)
			}
			if fork {
				searchOptions.SetFork(true)
			} else if noFork {
				searchOptions.SetFork(false)
			}
			if mirror {
				searchOptions.SetMirror(true)
			} else if noMirror {
				searchOptions.SetMirror(false)
			}
			if template {
				searchOptions.SetTemplate(true)
			} else if noTemplate {
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
	f.BoolVar(&archived, "archived", false, "Include only archived repositories")
	f.BoolVar(&noArchived, "no-archived", false, "Exclude archived repositories")
	f.BoolVar(&fork, "fork", false, "Include only forked repositories")
	f.BoolVar(&noFork, "no-fork", false, "Exclude forked repositories")
	f.BoolVar(&mirror, "mirror", false, "Include only mirrored repositories")
	f.BoolVar(&noMirror, "no-mirror", false, "Exclude mirrored repositories")
	f.BoolVar(&template, "is-template", false, "Include only template repositories")
	f.BoolVar(&noTemplate, "no-template", false, "Exclude template repositories")

	return cmd
}
