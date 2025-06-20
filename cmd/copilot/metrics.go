package copilot

import (
	"context"
	"fmt"
	"time"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type MetricsOptions struct {
	Org      string
	Since    string
	Until    string
	Exporter cmdutil.Exporter
}

func NewMetricsCmd() *cobra.Command {
	opts := &MetricsOptions{}
	cmd := &cobra.Command{
		Use:   "metrics <team-slug>",
		Short: "Show Copilot metrics for a team",
		Long:  "Show GitHub Copilot metrics for a specific team in an organization.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			team := args[0]
			var since, until *time.Time
			if opts.Since != "" {
				t, err := time.Parse(time.RFC3339, opts.Since)
				if err != nil {
					return fmt.Errorf("invalid since: %w", err)
				}
				since = &t
			}
			if opts.Until != "" {
				t, err := time.Parse(time.RFC3339, opts.Until)
				if err != nil {
					return fmt.Errorf("invalid until: %w", err)
				}
				until = &t
			}
			ctx := context.Background()
			repository, err := parser.Repository(parser.RepositoryOwner(opts.Org))
			if err != nil {
				return fmt.Errorf("failed to parse repository: %w", err)
			}
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			metrics, err := gh.GetCopilotTeamMetrics(ctx, client, repository.Owner, team, since, until)
			if err != nil {
				return fmt.Errorf("failed to get Copilot metrics: %w", err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderCopilotMetricsDefault(metrics)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&opts.Org, "org", "", "Organization name (required)")
	f.StringVar(&opts.Since, "since", "", "Start date (RFC3339, optional)")
	f.StringVar(&opts.Until, "until", "", "End date (RFC3339, optional)")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
