package mannequin

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// ListOptions holds format flags for the list command.
type ListOptions struct {
	Exporter cmdutil.Exporter
}

// NewListCmd creates a new cobra.Command for listing mannequins.
func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var nameOnly bool

	cmd := &cobra.Command{
		Use:     "list [owner]",
		Short:   "List mannequins in the organization",
		Long:    "List all mannequins (placeholder accounts for unclaimed users) in the specified organization.",
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner string
			if len(args) > 0 {
				owner = args[0]
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			mannequins, err := gh.ListMannequins(ctx, client, repository, nil)
			if err != nil {
				return fmt.Errorf("failed to list mannequins: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				renderer.RenderNames(mannequins)
			} else {
				renderer.RenderMannequinsDefault(mannequins)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only mannequin login names")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
