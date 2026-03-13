package emu

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

// NewListCmd creates a new cobra.Command for listing external groups.
func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var owner string
	var query string
	var fields []string

	cmd := &cobra.Command{
		Use:     "list [team-slug]",
		Short:   "List external groups in the organization or connected to a team",
		Long:    `List all external groups available in the organization, or list external groups connected to the specified team (Enterprise Managed Users).`,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[0])
				if err != nil {
					return fmt.Errorf("error parsing repository with team slug: %w", err)
				}

				ctx := context.Background()
				client, err := gh.NewGitHubClientWithRepo(repository)
				if err != nil {
					return fmt.Errorf("error creating GitHub client: %w", err)
				}

				groups, err := gh.ListExternalGroupsForTeam(ctx, client, repository, teamSlug)
				if err != nil {
					return fmt.Errorf("failed to list external groups for team '%s': %w", teamSlug, err)
				}

				renderer := render.NewRenderer(opts.Exporter)
				renderer.RenderExternalGroups(groups, fields)
				return nil
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

			groups, err := gh.ListExternalGroups(ctx, client, repository, query)
			if err != nil {
				return fmt.Errorf("failed to list external groups: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			renderer.RenderExternalGroups(groups, fields)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.StringVar(&query, "query", "", "Filter external groups by name (only applies when listing all groups)")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.ExternalGroupFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
