package access

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/internal/teamaccess"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

func NewCopyCmd() *cobra.Command {
	var owner string
	var includeInherited bool
	var effective bool
	var dryrun bool
	var exporter cmdutil.Exporter

	cmd := &cobra.Command{
		Use:   "copy <[[host/]owner/]src-team-slug> <[[host/]owner/]dst-team-slug>",
		Short: "Copy repository access from one team to another",
		Long: `Copy the source team's direct repository permissions, overwriting conflicting direct permissions without removing other destination grants.
Use --include-inherited to add inherited-only access that is missing or weaker at the destination, or --effective to copy source effective permissions.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcRepo, srcTeamSlug, err := parser.RepositoryWithTeamSlugs(args[0], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing source team: %w", err)
			}
			dstRepo, dstTeamSlug, err := parser.RepositoryWithTeamSlugs(args[1], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing destination team: %w", err)
			}
			srcClient, dstClient, err := gh.NewGitHubClientWith2Repos(srcRepo, dstRepo)
			if err != nil {
				return fmt.Errorf("error creating GitHub clients: %w", err)
			}
			mode := teamaccess.CopyDirect
			if includeInherited {
				mode = teamaccess.CopyWithInherited
			} else if effective {
				mode = teamaccess.CopyEffective
			}
			source := teamaccess.Team{Client: srcClient, Repository: srcRepo, Slug: srcTeamSlug}
			destination := teamaccess.Team{Client: dstClient, Repository: dstRepo, Slug: dstTeamSlug}
			plan, planErr := teamaccess.PlanCopy(cmd.Context(), source, destination, mode)
			if plan != nil {
				for _, warning := range plan.Warnings() {
					logger.Warn(warning, "from", args[0], "to", args[1])
				}
			}
			if dryrun {
				if planErr != nil {
					return fmt.Errorf("failed to plan copy of team repository permissions: %w", planErr)
				}
				if err := render.NewRenderer(exporter).RenderSyncChanges(plan.Changes(), nil); err != nil {
					return fmt.Errorf("failed to render planned team repository permission changes: %w", err)
				}
				return nil
			}
			if plan != nil {
				if err := teamaccess.ApplyCopy(cmd.Context(), plan); err != nil {
					return fmt.Errorf("failed to copy team repository permissions: %w", err)
				}
			}
			if planErr != nil {
				return fmt.Errorf("failed to copy team repository permissions: %w", planErr)
			}
			logger.Info("Team repository permissions copied.", "from", args[0], "to", args[1])
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")
	f.BoolVar(&includeInherited, "include-inherited", false, "Also copy source inherited permissions missing from the destination")
	f.BoolVar(&effective, "effective", false, "Copy source effective permissions, including direct-backed access")
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: show planned changes without applying them")
	cmd.MarkFlagsMutuallyExclusive("include-inherited", "effective")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
