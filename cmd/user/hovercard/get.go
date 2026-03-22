package hovercard

import (
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

			client, err := gh.NewGitHubClient()
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			hovercard, err := gh.GetUserHovercard(ctx, client, username, subjectType, subjectId)
			if err != nil {
				return fmt.Errorf("failed to get hovercard for user '%s': %w", username, err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			return renderer.RenderHovercard(hovercard)
		},
	}

	f := cmd.Flags()
	f.StringVar(&subjectType, "subject-type", "", "Type of subject for contextual hovercard")
	f.StringVar(&subjectId, "subject-id", "", "ID of subject for contextual hovercard")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
