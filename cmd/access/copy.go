package access

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

func NewCopyCmd() *cobra.Command {
	var owner string
	var force bool
	var dryrun bool
	var exporter cmdutil.Exporter

	cmd := &cobra.Command{
		Use:   "copy <[[host/]owner/]src-team-slug> <[[host/]owner/]dst-team-slug>",
		Short: "Copy repository access from one team to another",
		Long:  `Copy direct repository permissions from the source team to the destination team without removing other destination permissions.`,
		Args:  cobra.ExactArgs(2),
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
			if dryrun {
				plan, err := gh.PlanCopyTeamRepoPermissions(cmd.Context(), srcClient, srcRepo, srcTeamSlug, dstClient, dstRepo, dstTeamSlug, force)
				if err != nil {
					return fmt.Errorf("failed to plan copy of team repository permissions: %w", err)
				}
				for _, warning := range plan.Warnings() {
					logger.Warn(warning, "from", args[0], "to", args[1])
				}
				if err := render.NewRenderer(exporter).RenderSyncChanges(plan.Changes(), nil); err != nil {
					return fmt.Errorf("failed to render planned team repository permission changes: %w", err)
				}
				return nil
			}
			warnings, err := gh.CopyTeamRepoPermissions(cmd.Context(), srcClient, srcRepo, srcTeamSlug, dstClient, dstRepo, dstTeamSlug, force)
			for _, warning := range warnings {
				logger.Warn(warning, "from", args[0], "to", args[1])
			}
			if err != nil {
				return fmt.Errorf("failed to copy team repository permissions: %w", err)
			}
			logger.Info("Team repository permissions copied.", "from", args[0], "to", args[1])
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")
	f.BoolVarP(&force, "force", "f", false, "Overwrite existing destination permissions")
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: show planned changes without applying them")
	cmdutil.AddFormatFlags(cmd, &exporter)
	return cmd
}
