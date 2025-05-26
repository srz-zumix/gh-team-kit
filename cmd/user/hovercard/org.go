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

type OrgOptions struct {
	Exporter cmdutil.Exporter
}

func NewOrgCmd() *cobra.Command {
	opts := &OrgOptions{}
	var owner string
	cmd := &cobra.Command{
		Use:   "org [username]",
		Short: "Get organization hovercard for a user",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := ""
			if len(args) > 0 {
				username = args[1]
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

			org, err := gh.GetOrg(ctx, client, repository.Owner)
			if err != nil {
				return fmt.Errorf("failed to get organization '%s': %w", owner, err)
			}

			id := fmt.Sprintf("%d", *org.ID)
			hovercard, err := gh.GetUserHovercard(ctx, client, username, "organization", id)
			if err != nil {
				return fmt.Errorf("failed to get hovercard for user '%s': %w", username, err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderHovercard(hovercard)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization owner")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
