package member

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

// NewCopyCmd creates the `member copy` command for copying team members
func NewCopyCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "copy <[owner/]src-team-slug> <[owner/]dst-team-slug>",
		Short: "Copy members from source team to destination team (add only)",
		Long:  `Copy members from the source team to the destination team. Members in the source team will be added to the destination team, but no members will be removed from the destination team.`,
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
			client, err := gh.NewGitHubClientWithRepo(srcRepo)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}
			if err := gh.CopyTeamMembers(ctx, client, srcRepo, srcTeamSlug, dstRepo, dstTeamSlug); err != nil {
				return fmt.Errorf("failed to copy team members: %w", err)
			}
			fmt.Printf("Successfully copied members from %s to %s\n", srcTeam, dstTeam)
			return nil
		},
	}

	cmd.Flags().StringVar(&owner, "owner", "", "Default owner for team slugs (optional)")

	return cmd
}
