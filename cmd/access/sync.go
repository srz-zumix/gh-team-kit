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

func NewSyncCmd() *cobra.Command {
	var owner string
	var dryrun bool
	var includeInherited bool
	var effective bool
	var force bool
	var exporter cmdutil.Exporter

	cmd := &cobra.Command{
		Use:   "sync <[[host/]owner/]src-team-slug> <[[host/]owner/]dst-team-slug>",
		Short: "Synchronize repository access between teams",
		Long: `Synchronize repository permissions from the source team, removing destination direct grants outside the selected source access.
Permission downgrades, incomparable custom roles, and removals that may reduce effective access are skipped with a warning unless --force is set.
Use --include-inherited to synchronize inherited access based on the destination parent, or --effective to synchronize source effective permissions.`,
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
			plan, planErr := teamaccess.PlanSync(cmd.Context(), source, destination, mode, force)
			if dryrun || planErr != nil {
				if plan != nil {
					for _, warning := range plan.Warnings() {
						logger.Warn(warning, "from", args[0], "to", args[1])
					}
				}
				if planErr != nil {
					if !dryrun && plan != nil {
						if err := teamaccess.ApplySyncGrants(cmd.Context(), plan); err != nil {
							return fmt.Errorf("failed to sync team repository permissions: %w", err)
						}
					}
					return fmt.Errorf("failed to plan sync of team repository permissions: %w", planErr)
				}
				if err := render.NewRenderer(exporter).RenderSyncChanges(plan.Changes(), nil); err != nil {
					return fmt.Errorf("failed to render planned team repository permission changes: %w", err)
				}
				return nil
			}
			warnings, err := teamaccess.ApplySync(cmd.Context(), plan)
			for _, warning := range warnings {
				logger.Warn(warning, "from", args[0], "to", args[1])
			}
			if err != nil {
				return fmt.Errorf("failed to sync team repository permissions: %w", err)
			}
			logger.Info("Team repository permissions synced.", "from", args[0], "to", args[1])
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: show planned changes without applying them")
	f.BoolVar(&includeInherited, "include-inherited", false, "Synchronize source inherited permissions against destination parent access")
	f.BoolVar(&effective, "effective", false, "Synchronize source effective permissions, including direct-backed access")
	f.BoolVarP(&force, "force", "f", false, "Allow downgrades, incomparable custom roles, and removals that may reduce access")
	cmd.MarkFlagsMutuallyExclusive("include-inherited", "effective")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
