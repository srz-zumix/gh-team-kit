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

type TeamOptions struct {
	Exporter cmdutil.Exporter
}

func NewTeamsCmd() *cobra.Command {
	opts := &TeamOptions{}
	var nameOnly bool
	var owner string

	cmd := &cobra.Command{
		Use:     "teams [username]",
		Short:   "List teams of a user",
		Long:    `List all teams owned by the specified user`,
		Aliases: []string{"ls-team"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := ""
			if len(args) > 0 {
				username = args[0]
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

			renderer := render.NewRenderer(opts.Exporter)
			if owner == "" && username == "" {
				teams, err := gh.ListMyTeams(ctx, client)
				if err != nil {
					return fmt.Errorf("failed to list my teams: %w", err)
				}
				if nameOnly {
					renderer.RenderNames(teams)
				} else {
					headers := []string{"NAME", "DESCRIPTION", "MEMBER_COUNT", "REPOS_COUNT", "PRIVACY", "PARENT_SLUG", "ORGANIZATION", "URL"}
					renderer.RenderTeams(teams, headers)
				}
			} else {
				teams, err := gh.ListUserTeams(ctx, client, repository, username)
				if err != nil {
					return fmt.Errorf("failed to list teams for user '%s': %w", username, err)
				}
				if nameOnly {
					renderer.RenderNames(teams)
				} else {
					renderer.RenderTeamsDefault(teams)
				}
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only repository names")
	f.StringVar(&owner, "owner", "", "Specify the owner of the repository")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
