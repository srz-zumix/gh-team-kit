package org

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var owner string
	var nameOnly bool

	cmd := &cobra.Command{
		Use:   "list <org-role-name>",
		Short: "List teams assigned to a specific organization role",
		Long:  `Retrieve and display a list of all teams assigned to a specific role in the organization.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			role := args[0]
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			teams, err := gh.ListTeamsAssignedToRole(ctx, client, repository, role)
			if err != nil {
				return fmt.Errorf("failed to list teams assigned to role '%s': %w", role, err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if nameOnly {
				renderer.RenderNames(teams)
			} else {
				renderer.RenderTeamsDefault(teams)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&nameOnly, "name-only", "", false, "Output only team names")
	f.StringVarP(&owner, "owner", "", "", "Specify the organization name")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
