package collaborator

import (
	"fmt"
	"slices"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/pkg/userfilter"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// NewListCmd creates a new `user collaborator list` command
func NewListCmd() *cobra.Command {
	var details bool
	var exporter cmdutil.Exporter
	var fields []string
	var filter string
	var nameOnly bool
	var suspended cmdflags.MutuallyExclusiveBoolFlags

	cmd := &cobra.Command{
		Use:   "list [[HOST/]OWNER]",
		Short: "List outside collaborators of the organization",
		Long: `List all outside collaborators of the specified organization.
Use --filter to list only outside collaborators without two-factor authentication (2fa_disabled) or with insecure two-factor authentication methods (2fa_insecure).`,
		Aliases: []string{"ls"},
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var owner string
			if len(args) > 0 {
				owner = args[0]
			}

			fetchDetails := details
			if !nameOnly {
				// The listing API returns summary users only, so NAME/EMAIL require
				// full profile details and SUSPENDED requires suspension data.
				if slices.Contains(fields, "NAME") || slices.Contains(fields, "EMAIL") {
					fetchDetails = true
				}
				if fetchDetails || slices.Contains(fields, "SUSPENDED") {
					details = true
				}
			}
			if suspended.IsSet() {
				details = true
			}

			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			collaborators, err := gh.ListOutsideCollaborators(ctx, client, repository, filter)
			if err != nil {
				return fmt.Errorf("failed to list outside collaborators of organization %s: %w", repository.Owner, err)
			}

			if details {
				collaborators, err = userfilter.UpdateAndFilter(ctx, client, collaborators, suspended, fetchDetails, nameOnly)
				if err != nil {
					return fmt.Errorf("failed to update outside collaborators: %w", err)
				}
			}

			renderer := render.NewRenderer(exporter)
			if nameOnly {
				return renderer.RenderNames(collaborators)
			}
			if len(fields) > 0 {
				return renderer.RenderUsers(collaborators, fields)
			}
			if details {
				return renderer.RenderUsers(collaborators, []string{"LOGIN", "NAME", "EMAIL", "SUSPENDED"})
			}
			return renderer.RenderUsers(collaborators, []string{"USERNAME", "URL"})
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about outside collaborators")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.UserFieldList, "Fields to display")
	cmdutil.StringEnumFlag(cmd, &filter, "filter", "", "all", gh.OutsideCollaboratorFilterList, "Filter outside collaborators by two-factor authentication status")
	f.BoolVar(&nameOnly, "name-only", false, "Output only outside collaborator names")
	suspended.AddNoPrefixFlag(cmd, "suspended", "Output only suspended outside collaborators", "Exclude suspended outside collaborators")
	cmdutil.AddFormatFlags(cmd, &exporter)

	return cmd
}
