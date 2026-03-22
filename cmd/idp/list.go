package idp

import (
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

// NewListCmd creates a new cobra.Command for listing IDP groups.
func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var owner string
	var query string
	var fields []string

	cmd := &cobra.Command{
		Use:     "list [team-slug]",
		Short:   "List IDP groups in the organization or connected to a team",
		Long:    `List all IDP groups available in the organization, or list IDP groups connected to the specified team.`,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && query != "" {
				return fmt.Errorf("cannot use --query flag when team slug argument is provided")
			}

			if len(args) > 0 {
				repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[0])
				if err != nil {
					return fmt.Errorf("error parsing repository with team slug: %w", err)
				}

				client, err := gh.NewGitHubClientWithRepo(repository)
				if err != nil {
					return fmt.Errorf("error creating GitHub client: %w", err)
				}

				ctx := cmd.Context()
				groups, err := gh.ListIDPGroupsForTeam(ctx, client, repository, teamSlug)
				if err != nil {
					return fmt.Errorf("failed to list IDP groups for team '%s': %w", teamSlug, err)
				}

				renderer := render.NewRenderer(opts.Exporter)
				return renderer.RenderIDPGroups(groups, fields)
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			groups, err := gh.ListIDPGroups(ctx, client, repository, query)
			if err != nil {
				return fmt.Errorf("failed to list IDP groups: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			return renderer.RenderIDPGroups(groups, fields)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.StringVar(&query, "query", "", "Filter IDP groups by name (only applies when listing all groups)")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.IDPGroupFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
