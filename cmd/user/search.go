package user

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type SearchOptions struct {
	Exporter cmdutil.Exporter
}

// NewSearchCmd returns cobra.Command for searching users
func NewSearchCmd() *cobra.Command {
	opts := &SearchOptions{}
	var owner string
	var email string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search GitHub users",
		Long:  "Search for GitHub users by query string.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var query string
			if len(args) > 0 {
				query = args[0]
			}
			if email != "" {
				query += " in:email " + email
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
			findUsers, err := gh.SearchUsers(ctx, client, query)
			if err != nil {
				return fmt.Errorf("failed to search users: %w", err)
			}
			users, err := gh.UpdateUsers(ctx, client, findUsers)
			if err != nil {
				return fmt.Errorf("failed to update users: %w", err)
			}
			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderUserDetails(users)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.StringVar(&email, "email", "", "Filter users by email address")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)
	return cmd
}
