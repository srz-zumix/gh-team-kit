package member

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewSyncCmd creates the `member sync` command for synchronizing team members
func NewSyncCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "sync <[[host/]owner/]src-team-slug> <[[host/]owner/]dst-team-slug>",
		Short: "Sync members from source team to destination team",
		Long:  `Sync members from the source team to the destination team. Members in the source team will be added to the destination team, and members not in the source team will be removed from the destination team.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcTeam := args[0]
			dstTeam := args[1]
			srcRepo, srcTeamSlug, err := parser.RepositoryFromTeamSlugs(owner, srcTeam)
			if err != nil {
				return fmt.Errorf("error parsing source team: %w", err)
			}
			dstRepo, dstTeamSlug, err := parser.RepositoryFromTeamSlugs(owner, dstTeam)
			if err != nil {
				return fmt.Errorf("error parsing destination team: %w", err)
			}

			ctx := context.Background()
			srcClient, dstClient, err := gh.NewGitHubClientWith2Repos(srcRepo, dstRepo)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.SyncTeamMembers(ctx, srcClient, srcRepo, srcTeamSlug, dstClient, dstRepo, dstTeamSlug); err != nil {
				return fmt.Errorf("failed to sync team members: %w", err)
			}
			logger.Info("Members synced successfully.", "from", srcTeam, "to", dstTeam)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Default owner for team slugs")

	return cmd
}
