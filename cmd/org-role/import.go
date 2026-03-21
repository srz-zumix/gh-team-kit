package orgrole

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/ioutil"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewImportCmd creates a new cobra.Command for importing custom organization roles.
func NewImportCmd() *cobra.Command {
	var owner string
	var dryrun bool

	cmd := &cobra.Command{
		Use:   "import <input>",
		Short: "Import custom organization roles",
		Long: `Read a JSON list of custom organization roles (as produced by 'org-role list --format json') and create or update each role in the organization.
Each entry must have a "name" field. Existing roles with the same name are updated.
Specify '-' as input to read from stdin.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			roles, err := ioutil.DecodeJSONFile[[]*gh.CustomOrgRoles](input)
			if err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}

			if dryrun {
				logger.Info("Dry run completed. No changes were made.", "count", len(roles))
				return nil
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			var errs []error
			for _, role := range roles {
				if role.Name == nil {
					continue
				}
				// Only user-defined (Organization-sourced) roles can be created or updated.
				if role.GetSource() != "Organization" {
					logger.Info("Skipping non-Organization role.", "name", *role.Name, "source", role.GetSource())
					continue
				}
				_, err := gh.CreateOrUpdateOrgRole(ctx, client, repository, role.GetName(), role.GetDescription(), role.GetBaseRole(), role.Permissions)
				if err != nil {
					errs = append(errs, fmt.Errorf("failed to import org role '%s': %w", *role.Name, err))
				} else {
					logger.Info("Org role imported.", "name", *role.Name)
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

	return cmd
}
