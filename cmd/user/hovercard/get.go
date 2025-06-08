package hovercard

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type GetOptions struct {
	Exporter cmdutil.Exporter
}

func NewGetCmd() *cobra.Command {
	opts := &GetOptions{}
	var subjectType string
	var subjectId string

	cmd := &cobra.Command{
		Use:   "get [username]",
		Short: "Get hovercard for a user (no subject-type)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := ""
			if len(args) > 0 {
				username = args[0]
			}
			ctx := context.Background()
			client, err := gh.NewGitHubClient()
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}
			hovercard, err := gh.GetUserHovercard(ctx, client, username, subjectType, subjectId)
			if err != nil {
				return fmt.Errorf("failed to get hovercard for user '%s': %w", username, err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderHovercard(hovercard)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&subjectType, "subject-type", "", "Type of subject for contextual hovercard")
	f.StringVar(&subjectId, "subject-id", "", "ID of subject for contextual hovercard")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
