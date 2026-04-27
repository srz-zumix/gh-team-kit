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
	"github.com/srz-zumix/go-gh-extension/pkg/settings"
)

// NewImportCmd creates a new cobra.Command for importing users into an organization.
func NewImportCmd() *cobra.Command {
	var owner string
	var dryrun bool
	var defaultRole string
	var mapFile string
	var ignoreErrors bool

	cmd := &cobra.Command{
		Use:   "import <input>",
		Short: "Import users into the organization",
		Long: `Read a JSON list of users (as produced by 'user list --format json') and add each user to the organization.
Entries without a "login" field are skipped. The role is taken from the "role_name" field if present; otherwise --role is used as default.
When --usermap is specified, source logins are automatically converted to target logins using the mapping file (as produced by 'user map').
Specify '-' as input to read from stdin.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]

			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			users, err := ioutil.DecodeJSONFile[[]*gh.GitHubUser](input)
			if err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}

			// Load mapping file if specified
			var compiledMappings *settings.CompiledMappings
			if mapFile != "" {
				compiledMappings, err = settings.NewCompiledMappingsFromFile(mapFile)
				if err != nil {
					return fmt.Errorf("error loading mapping file '%s': %w", mapFile, err)
				}
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

				// Apply mapping if available
				login := *u.Login
				if compiledMappings != nil {
					if targetLogin, ok := compiledMappings.ResolveSrc(login); ok {
						login = targetLogin
					}
				}

				role := u.GetRoleName()
				if role == "" {
					role = defaultRole
				}
				_, err := gh.AddOrUpdateOrgMember(ctx, client, repository, login, role)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to add user '%s': %w", login, err))
				} else {
					logger.Info("User added to organization.", "username", login, "role", role)
				}
			}

			if len(errs) > 0 {
				if !ignoreErrors {
					return fmt.Errorf("encountered errors during import: %w", errors.Join(errs...))
				}
				logger.Warn(fmt.Sprintf("encountered errors during import: %v", errors.Join(errs...)))
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization ([HOST/]OWNER)")
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: do not actually apply changes")
	f.StringVar(&mapFile, "usermap", "", "User mapping file (as produced by 'user map') for login conversion during import")
	cmdutil.StringEnumFlag(cmd, &defaultRole, "role", "", gh.TeamMembershipRoleMember, gh.OrgMembershipList, "Default role when not specified in input (member or admin)")
	f.BoolVar(&ignoreErrors, "ignore-errors", false, "Continue without exiting on error during import")

	return cmd
}
