package role

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var nameOnly bool

	cmd := &cobra.Command{
		Use:   "list [owner]",
		Short: "List roles in the organization",
		Long:  `List all roles available in the organization.`,
		Args:  cobra.MaximumNArgs(1),
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

			roles, err := gh.ListOrgRoles(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to list roles for owner '%s': %w", owner, err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				renderer.RenderNames(roles)
			} else {
				renderer.RenderCustomOrgRoles(roles)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&nameOnly, "name-only", false, "Output only role names")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
