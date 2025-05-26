package hovercard

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type RepoOptions struct {
	Exporter cmdutil.Exporter
}

func NewRepoCmd() *cobra.Command {
	opts := &RepoOptions{}
	var repo string
	cmd := &cobra.Command{
		Use:   "repo [username]",
		Short: "Get repository hovercard for a user",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := ""
			if len(args) > 0 {
				username = args[0]
			}

			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			r, err := gh.GetRepository(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to get repository '%s': %w", repo, err)
			}

			id := fmt.Sprintf("%d", *r.ID)
			hovercard, err := gh.GetUserHovercard(ctx, client, username, "repository", id)
			if err != nil {
				return fmt.Errorf("failed to get hovercard for user '%s': %w", username, err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderHovercard(hovercard)
			return nil
		},
	}
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "Repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
