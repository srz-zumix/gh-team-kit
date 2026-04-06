package member

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewCopyCmd creates the `member copy` command for copying team members
func NewCopyCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "copy <[[host/]owner/]src-team-slug> <[[host/]owner/]dst-team-slug>",
		Short: "Copy members from source team to destination team (add only)",
		Long:  `Copy members from the source team to the destination team. Members in the source team will be added to the destination team, but no members will be removed from the destination team.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcTeam := args[0]
			dstTeam := args[1]
			srcRepo, srcTeamSlug, err := parser.RepositoryWithTeamSlugs(srcTeam, parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing source team: %w", err)
			}
			dstRepo, dstTeamSlug, err := parser.RepositoryWithTeamSlugs(dstTeam, parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing destination team: %w", err)
			}

			srcClient, dstClient, err := gh.NewGitHubClientWith2Repos(srcRepo, dstRepo)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			if err := gh.CopyTeamMembers(ctx, srcClient, srcRepo, srcTeamSlug, dstClient, dstRepo, dstTeamSlug); err != nil {
				return fmt.Errorf("failed to copy team members: %w", err)
			}
			logger.Info("Members copied successfully.", "from", srcTeam, "to", dstTeam)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")

	return cmd
}
