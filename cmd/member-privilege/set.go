package memberprivilege

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/google/go-github/v84/github"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewSetCmd creates a command to update the member privileges of an organization.
func NewSetCmd() *cobra.Command {
	var owner string
	var defaultRepoPermission string
	var membersCanCreateRepos cmdflags.MutuallyExclusiveBoolFlags
	var membersCanCreatePublicRepos cmdflags.MutuallyExclusiveBoolFlags
	var membersCanCreatePrivateRepos cmdflags.MutuallyExclusiveBoolFlags
	var membersCanCreateInternalRepos cmdflags.MutuallyExclusiveBoolFlags
	var membersCanForkPrivateRepos cmdflags.MutuallyExclusiveBoolFlags
	var membersCanCreatePages cmdflags.MutuallyExclusiveBoolFlags
	var membersCanCreatePublicPages cmdflags.MutuallyExclusiveBoolFlags
	var membersCanCreatePrivatePages cmdflags.MutuallyExclusiveBoolFlags
	var membersCanCreateTeams cmdflags.MutuallyExclusiveBoolFlags
	var webCommitSignoffRequired cmdflags.MutuallyExclusiveBoolFlags

	cmd := &cobra.Command{
		Use:   "set [owner]",
		Short: "Set member privileges of an organization",
		Long:  `Update one or more member privileges settings of the specified organization.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				owner = args[0]
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			input := &github.Organization{}
			if cmd.Flags().Changed("default-repo-permission") {
				input.DefaultRepoPermission = &defaultRepoPermission
			}
			input.MembersCanCreateRepos = membersCanCreateRepos.GetValue()
			input.MembersCanCreatePublicRepos = membersCanCreatePublicRepos.GetValue()
			input.MembersCanCreatePrivateRepos = membersCanCreatePrivateRepos.GetValue()
			input.MembersCanCreateInternalRepos = membersCanCreateInternalRepos.GetValue()
			input.MembersCanForkPrivateRepos = membersCanForkPrivateRepos.GetValue()
			input.MembersCanCreatePages = membersCanCreatePages.GetValue()
			input.MembersCanCreatePublicPages = membersCanCreatePublicPages.GetValue()
			input.MembersCanCreatePrivatePages = membersCanCreatePrivatePages.GetValue()
			input.MembersCanCreateTeams = membersCanCreateTeams.GetValue()
			input.WebCommitSignoffRequired = webCommitSignoffRequired.GetValue()

			ctx := cmd.Context()
			_, err = gh.EditOrgMemberPrivileges(ctx, client, repository, input)
			if err != nil {
				return fmt.Errorf("failed to update member privileges: %w", err)
			}

			logger.Info("Member privileges updated successfully.", "org", repository.Owner)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.StringEnumFlag(cmd, &defaultRepoPermission, "default-repo-permission", "", "", gh.OrgDefaultRepoPermissionList, "Default repository permission for organization members")
	membersCanCreateRepos.AddNoPrefixFlag(cmd, "members-can-create-repos", "Allow members to create repositories", "Disallow members from creating repositories")
	membersCanCreatePublicRepos.AddNoPrefixFlag(cmd, "members-can-create-public-repos", "Allow members to create public repositories", "Disallow members from creating public repositories")
	membersCanCreatePrivateRepos.AddNoPrefixFlag(cmd, "members-can-create-private-repos", "Allow members to create private repositories", "Disallow members from creating private repositories")
	membersCanCreateInternalRepos.AddNoPrefixFlag(cmd, "members-can-create-internal-repos", "Allow members to create internal repositories", "Disallow members from creating internal repositories")
	membersCanForkPrivateRepos.AddNoPrefixFlag(cmd, "members-can-fork-private-repos", "Allow members to fork private repositories", "Disallow members from forking private repositories")
	membersCanCreatePages.AddNoPrefixFlag(cmd, "members-can-create-pages", "Allow members to create GitHub Pages sites", "Disallow members from creating GitHub Pages sites")
	membersCanCreatePublicPages.AddNoPrefixFlag(cmd, "members-can-create-public-pages", "Allow members to create public GitHub Pages sites", "Disallow members from creating public GitHub Pages sites")
	membersCanCreatePrivatePages.AddNoPrefixFlag(cmd, "members-can-create-private-pages", "Allow members to create private GitHub Pages sites", "Disallow members from creating private GitHub Pages sites")
	membersCanCreateTeams.AddNoPrefixFlag(cmd, "members-can-create-teams", "Allow members to create teams", "Disallow members from creating teams")
	webCommitSignoffRequired.AddNoPrefixFlag(cmd, "web-commit-signoff-required", "Require web commit signoff", "Do not require web commit signoff")

	return cmd
}
