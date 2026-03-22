package emu

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

// NewListCmd creates a new cobra.Command for listing external groups.
func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var owner string
	var query string
	var fields []string
	var details bool

	cmd := &cobra.Command{
		Use:     "list [team-slug]",
		Short:   "List external groups in the organization or connected to a team",
		Long:    `List all external groups available in the organization, or list external groups connected to the specified team (Enterprise Managed Users).`,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := ""
			if len(args) > 0 {
				teamSlug = args[0]
			}

			repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, teamSlug)
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			groups, err := gh.SearchExternalGroups(ctx, client, repository, query, teamSlug)
			if err != nil {
				return fmt.Errorf("failed to list external groups: %w", err)
			}

			if details {
				groups, err = gh.GetExternalGroupDetails(ctx, client, repository, groups)
				if err != nil {
					return fmt.Errorf("failed to get external group details: %w", err)
				}
			}

			renderer := render.NewRenderer(opts.Exporter)
			if details {
				return renderer.RenderExternalGroupsDetails(groups, fields)
			} else {
				return renderer.RenderExternalGroups(groups, fields)
			}
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.StringVar(&query, "query", "", "Filter external groups by name")
	f.BoolVar(&details, "details", false, "Fetch detailed info (teams/members) for each group")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.ExternalGroupFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
