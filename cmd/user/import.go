package user

import (
	"errors"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/ioutil"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewImportCmd creates a new cobra.Command for importing users into an organization.
func NewImportCmd() *cobra.Command {
	var owner string
	var dryrun bool
	var defaultRole string

	cmd := &cobra.Command{
		Use:   "import <input>",
		Short: "Import users into the organization",
		Long: `Read a JSON list of users (as produced by 'user list --format json') and add each user to the organization.
Entries without a "login" field are skipped. The role is taken from the "role_name" field if present; otherwise --role is used as default.
Specify '-' as input to read from stdin.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			users, err := ioutil.DecodeJSONFile[[]*gh.GitHubUser](input)
			if err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}

			if dryrun {
				logger.Info("Dry run completed. No changes were made.", "count", len(users))
				return nil
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			var errs []error
			for _, u := range users {
				if u.Login == nil {
					continue
				}
				role := u.GetRoleName()
				if role == "" {
					role = defaultRole
				}
				_, err := gh.AddOrUpdateOrgMember(ctx, client, repository, *u.Login, role)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to add user '%s': %w", *u.Login, err))
				} else {
					logger.Info("User added to organization.", "username", *u.Login, "role", role)
				}
			}

			if len(errs) > 0 {
				return fmt.Errorf("encountered errors during import: %w", errors.Join(errs...))
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: do not actually apply changes")
	cmdutil.StringEnumFlag(cmd, &defaultRole, "role", "", gh.TeamMembershipRoleMember, gh.OrgMembershipList, "Default role when not specified in input (member or admin)")

	return cmd
}
