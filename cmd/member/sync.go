package member

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

// NewSyncCmd creates the `member sync` command for synchronizing team members
func NewSyncCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "sync <[owner/]src-team-slug> <[owner/]dst-team-slug>",
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

			if srcRepo.Host != dstRepo.Host {
				return fmt.Errorf("source and destination teams must be on the same host (%s != %s)", srcRepo.Host, dstRepo.Host)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(srcRepo)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}
			if err := gh.SyncTeamMembers(ctx, client, srcRepo, srcTeamSlug, dstRepo, dstTeamSlug); err != nil {
				return fmt.Errorf("failed to sync team members: %w", err)
			}
			fmt.Printf("Successfully synced members from %s to %s\n", srcTeam, dstTeam)
			return nil
		},
	}

	cmd.Flags().StringVar(&owner, "owner", "", "Default owner for team slugs (optional)")

	return cmd
}
