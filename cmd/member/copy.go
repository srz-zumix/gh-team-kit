package member

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
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
			srcClient, dstClient, err := gh.NewGitHubClientWith2Repos(srcRepo, dstRepo)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.CopyTeamMembers(ctx, srcClient, srcRepo, srcTeamSlug, dstClient, dstRepo, dstTeamSlug); err != nil {
				return fmt.Errorf("failed to copy team members: %w", err)
			}
			fmt.Printf("Successfully copied members from %s to %s\n", srcTeam, dstTeam)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Default owner for team slugs")

	return cmd
}
