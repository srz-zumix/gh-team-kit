package memberprivilege

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewCopyCmd creates a command to copy member privileges from one organization to another.
func NewCopyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copy <src-owner> <dst-owner>",
		Short: "Copy member privileges from one organization to another",
		Long:  `Copy member privileges settings from the source organization to the destination organization.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			srcOwner := args[0]
			dstOwner := args[1]

			srcRepo, err := parser.Repository(parser.RepositoryOwner(srcOwner))
			if err != nil {
				return fmt.Errorf("error parsing source repository: %w", err)
			}
			dstRepo, err := parser.Repository(parser.RepositoryOwner(dstOwner))
			if err != nil {
				return fmt.Errorf("error parsing destination repository: %w", err)
			}

			srcClient, dstClient, err := gh.NewGitHubClientWith2Repos(srcRepo, dstRepo)
			if err != nil {
				return fmt.Errorf("error creating GitHub clients: %w", err)
			}

			ctx := cmd.Context()
			src, err := gh.GetOrgMemberPrivileges(ctx, srcClient, srcRepo)
			if err != nil {
				return fmt.Errorf("failed to get member privileges from source organization '%s': %w", srcOwner, err)
			}

			input := &gh.Organization{
				DefaultRepoPermission:         src.DefaultRepoPermission,
				MembersCanCreateRepos:         src.MembersCanCreateRepos,
				MembersCanCreatePublicRepos:   src.MembersCanCreatePublicRepos,
				MembersCanCreatePrivateRepos:  src.MembersCanCreatePrivateRepos,
				MembersCanCreateInternalRepos: src.MembersCanCreateInternalRepos,
				MembersCanForkPrivateRepos:    src.MembersCanForkPrivateRepos,
				MembersCanCreatePages:         src.MembersCanCreatePages,
				MembersCanCreatePublicPages:   src.MembersCanCreatePublicPages,
				MembersCanCreatePrivatePages:  src.MembersCanCreatePrivatePages,
				MembersCanCreateTeams:         src.MembersCanCreateTeams,
				WebCommitSignoffRequired:      src.WebCommitSignoffRequired,
			}

			_, err = gh.EditOrg(ctx, dstClient, dstRepo, input)
			if err != nil {
				return fmt.Errorf("failed to apply member privileges to destination organization '%s': %w", dstOwner, err)
			}

			logger.Info("Member privileges copied successfully.", "from", srcOwner, "to", dstOwner)
			return nil
		},
	}

	return cmd
}
