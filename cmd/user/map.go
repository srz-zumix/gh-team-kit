package user

import (
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
	"github.com/srz-zumix/go-gh-extension/pkg/settings"
)

// MapOptions holds options for the map command.
type MapOptions struct {
	Exporter cmdutil.Exporter
}

// NewMapCmd creates a new cobra.Command for creating user mappings between two hosts.
func NewMapCmd() *cobra.Command {
	opts := &MapOptions{}
	var owner string
	var output string
	var format string
	var all bool
	var quiet bool
	var emu bool
	var noSuspended bool

	cmd := &cobra.Command{
		Use:   "map <target>",
		Short: "Create a user mapping file between source and target hosts",
		Long: `Generate a YAML mapping file that correlates users by their public email between a source and target organization.
This mapping can be used with 'user import --usermap' to automatically convert source logins to target logins during import.

The command retrieves all users from both the source organization (via --owner) and target organization (positional argument),
matching them by public email address.

Both --owner and <target> accept the "[HOST/]OWNER" format.
With --all, source users that could not be matched by email are also included with an empty dst field.

Example:
  gh team-kit user map myorg --owner github.example.com/myorg --output user-map.yaml
  gh team-kit user map github.example.com/myorg --owner myorg --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]

			targetRepository, err := parser.Repository(parser.RepositoryOwnerWithHost(target))
			if err != nil {
				return fmt.Errorf("error parsing target: %w", err)
			}

			srcRepository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing owner: %w", err)
			}

			srcClient, dstClient, err := gh.NewGitHubClientWith2Repos(srcRepository, targetRepository)
			if err != nil {
				return fmt.Errorf("error creating GitHub clients: %w", err)
			}

			ctx := cmd.Context()
			srcUsers, err := gh.ListOrgMembers(ctx, srcClient, srcRepository, []string{}, false)
			if err != nil {
				return fmt.Errorf("failed to list members on source '%s': %w", parser.GetRepositoryFullNameWithHost(srcRepository), err)
			}
			srcUsers, err = gh.UpdateUsers(ctx, srcClient, srcUsers)
			if err != nil {
				return fmt.Errorf("failed to fetch user details on source '%s': %w", parser.GetRepositoryFullNameWithHost(srcRepository), err)
			}
			if noSuspended {
				srcUsers = gh.ExcludeSuspendedUsers(srcUsers)
			}

			dstUsers, err := gh.ListOrgMembers(ctx, dstClient, targetRepository, []string{}, false)
			if err != nil {
				return fmt.Errorf("failed to list members on target '%s': %w", parser.GetRepositoryFullNameWithHost(targetRepository), err)
			}
			dstUsers, err = gh.UpdateUsers(ctx, dstClient, dstUsers)
			if err != nil {
				return fmt.Errorf("failed to fetch user details on target '%s': %w", parser.GetRepositoryFullNameWithHost(targetRepository), err)
			}
			if noSuspended {
				dstUsers = gh.ExcludeSuspendedUsers(dstUsers)
			}

			emailToTargetLogin := make(map[string]string)
			for _, dstUser := range dstUsers {
				if dstUser.Email != nil && *dstUser.Email != "" {
					emailToTargetLogin[*dstUser.Email] = *dstUser.Login
				}
			}

			var mappings []settings.UserMapping
			for _, srcUser := range srcUsers {
				var email string
				if srcUser.Email != nil {
					email = *srcUser.Email
				}

				targetLogin, exists := emailToTargetLogin[email]
				if !exists {
					if !quiet {
						logger.Warn("No matching target user found for email", "email", email, "src", *srcUser.Login)
					}
					if !all {
						continue
					}
				}

				mappings = append(mappings, settings.UserMapping{
					Src:   *srcUser.Login,
					Dst:   targetLogin,
					Email: email,
				})
			}

			if emu {
				mappings = settings.CompactEMUMappings(mappings)
			}

			if output != "" {
				if _, err := settings.Write(output, mappings); err != nil {
					return err
				}
				logger.Info("User mapping file created successfully", "file", output, "count", len(mappings))
				return nil
			}

			renderer := render.NewRenderer(opts.Exporter)
			if format == "yaml" {
				return renderer.RenderUserMappingsYAML(mappings)
			}
			return renderer.RenderUserMappings(mappings, nil)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Source organization ([HOST/]OWNER; uses current repository's organization if omitted)")
	f.StringVarP(&output, "output", "o", "", "Write mapping YAML to this file; if omitted, output goes to stdout using --format (default: table)")
	f.BoolVarP(&all, "all", "a", false, "Include source users that could not be matched by email (dst will be empty)")
	f.BoolVar(&quiet, "quiet", false, "Suppress warnings for source users with no matching target user")
	f.BoolVar(&emu, "emu", false, "Compact matched pairs sharing the same base login into regex entries (e.g. (.+)_srcslug → $1_dstslug)")
	f.BoolVar(&noSuspended, "no-suspended", false, "Exclude suspended users from source and target before matching")

	_ = cmdflags.AddFormatFlags(cmd, &opts.Exporter, &format, "", []string{"yaml"})

	cmd.MarkFlagsMutuallyExclusive("output", "format")

	return cmd
}
