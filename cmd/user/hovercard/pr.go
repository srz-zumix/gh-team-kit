package hovercard

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type PrOptions struct {
	Exporter cmdutil.Exporter
}

func NewPrCmd() *cobra.Command {
	opts := &PrOptions{}
	var repo string
	cmd := &cobra.Command{
		Use:   "pr <pr-number> [username]",
		Short: "Get pull request hovercard for a user",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prNumber := args[0]
			username := ""
			if len(args) > 1 {
				username = args[1]
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

			pr, err := gh.GetPullRequest(ctx, client, repository, prNumber)
			if err != nil {
				return fmt.Errorf("failed to get pull request '%s' in repository '%s': %w", prNumber, repo, err)
			}

			id := fmt.Sprintf("%d", *pr.ID)
			hovercard, err := gh.GetUserHovercard(ctx, client, username, "pull_request", id)
			if err != nil {
				return fmt.Errorf("failed to get hovercard for user '%s': %w", username, err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderHovercard(hovercard)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "Repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
