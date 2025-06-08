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

type IssueOptions struct {
	Exporter cmdutil.Exporter
}

func NewIssueCmd() *cobra.Command {
	opts := &IssueOptions{}
	var repo string
	cmd := &cobra.Command{
		Use:   "issue <issue-number> [username]",
		Short: "Get issue hovercard for a user",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueNumber := args[0]
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

			issue, err := gh.GetIssue(ctx, client, repository, issueNumber)
			if err != nil {
				return fmt.Errorf("failed to get issue '%s' in repository '%s': %w", issueNumber, repo, err)
			}

			id := fmt.Sprintf("%d", *issue.ID)
			hovercard, err := gh.GetUserHovercard(ctx, client, username, "issue", id)
			if err != nil {
				return fmt.Errorf("failed to get hovercard for user '%s': %w", username, err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderHovercard(hovercard)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "Repository in the format 'owner/repo' (used with --issue-number)")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
