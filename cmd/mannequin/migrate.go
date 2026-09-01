package mannequin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/settings"
)

// NewMigrateCmd creates a new cobra.Command for bulk-migrating mannequins using a user mapping file.
// It lists all mannequins in the organization and reattributes each one whose login or email
// is found in the mapping file.
func NewMigrateCmd() *cobra.Command {
	var owner string
	var srcOwner string
	var mapFile string
	var skipInvitation bool
	var force bool
	var dryrun bool
	var noSuspended bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Bulk-migrate mannequins using a user mapping file",
		Long: `List all mannequins in the organization and reattribute each one to its mapped target user.

The mapping file (--usermap) must be a YAML file as produced by 'user map'.
Each mannequin is matched to a mapping entry first by src login, then by email.
Mannequins already claimed are skipped unless --force is specified.
Entries whose dst login is empty are skipped.
Bot accounts (mannequin login ending with '[bot]') are skipped because they cannot be reclaimed.
With --src, mannequins that are not members of the source organization are skipped without error when the target user cannot be found.
With --no-suspended, --src is required; mannequins whose login is a suspended member of the source organization are skipped.
Processing continues on per-mannequin errors; all collected errors are reported at the end.

Example:
  gh team-kit mannequin migrate --owner myorg --usermap user-map.yaml
  gh team-kit mannequin migrate --owner myorg --usermap user-map.yaml --skip-invitation --dryrun
  gh team-kit mannequin migrate --owner myorg --usermap user-map.yaml --src ghes.example.com/srcorg
  gh team-kit mannequin migrate --owner myorg --usermap user-map.yaml --no-suspended --src ghes.example.com/srcorg`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noSuspended && srcOwner == "" {
				return fmt.Errorf("--no-suspended requires --src")
			}

			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := cmd.Context()

			// Load and compile usermap before performing any API calls so that an
			// invalid/missing mapping file fails fast without wasting source-org lookups.
			compiledMappings, err := settings.NewCompiledMappingsFromFile(mapFile)
			if err != nil {
				return fmt.Errorf("error loading mapping file: %w", err)
			}

			var client *gh.GitHubClient
			var srcLogins map[string]struct{}
			var suspendedSrcLogins map[string]struct{}
			if srcOwner != "" {
				srcRepository, err := parser.Repository(parser.RepositoryOwnerWithHost(srcOwner))
				if err != nil {
					return fmt.Errorf("error parsing src: %w", err)
				}

				var srcClient *gh.GitHubClient
				client, srcClient, err = gh.NewGitHubClientWith2Repos(repository, srcRepository)
				if err != nil {
					return fmt.Errorf("error creating GitHub clients: %w", err)
				}

				srcMembers, err := gh.ListOrgMembers(ctx, srcClient, srcRepository, []string{}, false)
				if err != nil {
					return fmt.Errorf("failed to list members on source organization '%s': %w", parser.GetRepositoryFullNameWithHost(srcRepository), err)
				}
				srcLogins = make(map[string]struct{})
				for _, u := range srcMembers {
					if u.Login != nil {
						srcLogins[strings.ToLower(*u.Login)] = struct{}{}
					}
				}
				if noSuspended {
					srcMembers, err = gh.UpdateUsers(ctx, srcClient, srcMembers)
					if err != nil {
						return fmt.Errorf("failed to fetch member details on source organization '%s': %w", parser.GetRepositoryFullNameWithHost(srcRepository), err)
					}
					suspendedSrcLogins = make(map[string]struct{})
					for _, u := range gh.CollectSuspendedUsers(srcMembers) {
						if u.Login != nil {
							suspendedSrcLogins[strings.ToLower(*u.Login)] = struct{}{}
						}
					}
				}
			} else {
				client, err = gh.NewGitHubClientWithRepo(repository)
				if err != nil {
					return fmt.Errorf("error creating GitHub client: %w", err)
				}
			}

			// List all mannequins in the organization
			mannequins, err := gh.ListMannequins(ctx, client, repository, nil)
			if err != nil {
				return fmt.Errorf("failed to list mannequins: %w", err)
			}

			// Get organization node ID (needed for attribution APIs)
			org, err := gh.GetOrg(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to get organization '%s': %w", repository.Owner, err)
			}
			if org.NodeID == nil {
				return fmt.Errorf("failed to get node ID for organization '%s'", repository.Owner)
			}
			orgNodeID := *org.NodeID

			var errs []error
			for i := range mannequins {
				m := &mannequins[i]
				mannequinLogin := string(m.Login)

				// Skip already-claimed mannequins unless --force
				if !force && string(m.Claimant.Login) != "" {
					logger.Info("Skipping already claimed mannequin", "mannequin", mannequinLogin, "claimant", string(m.Claimant.Login))
					continue
				}

				// Bot accounts cannot be reattributed via mannequin reclamation, skip them.
				// Detect bots by the mannequin's own login so that usermap replacements
				// (e.g. EMU suffix rules) do not hide the '[bot]' suffix from this check.
				if strings.HasSuffix(mannequinLogin, "[bot]") {
					logger.Warn("Mannequin is a bot, skipping", "mannequin", mannequinLogin)
					continue
				}

				// Skip mannequins whose login is a suspended member of the source organization
				// when --no-suspended is set. The mannequin login corresponds to the account
				// on the source organization, so suspension is checked there rather than on
				// the mapped target login.
				if noSuspended {
					if _, ok := suspendedSrcLogins[strings.ToLower(mannequinLogin)]; ok {
						logger.Warn("Mannequin is a suspended member on the source organization, skipping", "mannequin", mannequinLogin)
						continue
					}
				}

				// Find matching mapping entry: prefer src-login match (with regex), fallback to email
				var targetLogin string
				var found bool
				if dst, ok := compiledMappings.ResolveSrc(mannequinLogin); ok {
					targetLogin = dst
					found = true
				} else if m.Email != nil && string(*m.Email) != "" {
					if entry, ok := compiledMappings.ResolveEmail(string(*m.Email)); ok {
						targetLogin = entry.Dst
						found = true
					}
				}

				if !found {
					logger.Warn("No mapping found for mannequin, skipping", "mannequin", mannequinLogin)
					continue
				}
				if targetLogin == "" {
					logger.Warn("Mapping dst is empty, skipping", "mannequin", mannequinLogin)
					continue
				}

				if dryrun {
					logger.Info("Would reattribute mannequin", "mannequin", mannequinLogin, "target-user", targetLogin)
					continue
				}

				mannequinNodeID := m.NodeID()

				// Get target user node ID
				targetUser, err := gh.FindUser(ctx, client, targetLogin)
				if err != nil {
					// A mannequin that is not a member of the source organization is out of scope
					// for this migration, so a missing target user is not treated as an error.
					if srcLogins != nil {
						if _, ok := srcLogins[strings.ToLower(mannequinLogin)]; !ok {
							logger.Debug("Target user not found and mannequin is not a member of the source organization, skipping", "mannequin", mannequinLogin, "target-user", targetLogin, "error", err)
							continue
						}
					}
					logger.Error("Failed to find target user, skipping", "mannequin", mannequinLogin, "target-user", targetLogin, "error", err)
					errs = append(errs, fmt.Errorf("failed to find user '%s' for mannequin '%s': %w", targetLogin, mannequinLogin, err))
					continue
				}
				if targetUser.NodeID == nil {
					logger.Error("Failed to get node ID for target user, skipping", "mannequin", mannequinLogin, "target-user", targetLogin)
					errs = append(errs, fmt.Errorf("failed to get node ID for user '%s' for mannequin '%s'", targetLogin, mannequinLogin))
					continue
				}

				targetUserNodeID := targetUser.GetNodeID()

				if skipInvitation {
					if err := gh.ReattributeMannequinToUser(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
						logger.Error("Failed to reattribute mannequin, skipping", "mannequin", mannequinLogin, "target-user", targetLogin, "error", err)
						errs = append(errs, fmt.Errorf("failed to reattribute mannequin '%s' to user '%s': %w", mannequinLogin, targetLogin, err))
						continue
					}
					logger.Info("Mannequin reattributed successfully.", "mannequin", mannequinLogin, "target-user", targetLogin)
				} else {
					if err := gh.CreateAttributionInvitation(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
						logger.Error("Failed to invite user to claim mannequin, skipping", "mannequin", mannequinLogin, "target-user", targetLogin, "error", err)
						errs = append(errs, fmt.Errorf("failed to invite user '%s' to claim mannequin '%s': %w", targetLogin, mannequinLogin, err))
						continue
					}
					logger.Info("Attribution invitation sent.", "mannequin", mannequinLogin, "target-user", targetLogin)
				}
			}
			if len(errs) > 0 {
				return fmt.Errorf("encountered errors during mannequin migration: %w", errors.Join(errs...))
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Target organization ([HOST/]OWNER; uses current repository's organization if omitted)")
	f.StringVar(&srcOwner, "src", "", "Source organization ([HOST/]OWNER) whose members are used to scope mannequins; required when --no-suspended is specified")
	f.StringVar(&mapFile, "usermap", "", "User mapping file (as produced by 'user map') for login resolution")
	f.BoolVar(&skipInvitation, "skip-invitation", false, "Skip the invitation step and directly reclaim mannequins (requires the feature to be enabled by GitHub Support)")
	f.BoolVar(&force, "force", false, "Process even mannequins that are already claimed")
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: show what would be done without making changes")
	f.BoolVar(&noSuspended, "no-suspended", false, "Skip mannequins whose login is a suspended member of --src")
	if err := cmd.MarkFlagRequired("usermap"); err != nil {
		panic(err)
	}

	return cmd
}
