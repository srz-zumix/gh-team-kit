package codereview

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type GetOptions struct {
	Exporter cmdutil.Exporter
}

func NewGetCmd() *cobra.Command {
	opts := &GetOptions{}
	var owner string

	var cmd = &cobra.Command{
		Use:   "get <team-slug>",
		Short: "Get code reviews settings",
		Long:  `Get code reviews settings.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[0])
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			s, err := gh.GetTeamCodeReviewSettings(ctx, client, repository, teamSlug)
			if err != nil {
				return fmt.Errorf("failed to get code review settings: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderTeamCodeReviewSettingsDefault(s)

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
