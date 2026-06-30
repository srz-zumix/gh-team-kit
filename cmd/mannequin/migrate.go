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
	var mapFile string
	var skipInvitation bool
	var force bool
	var dryrun bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Bulk-migrate mannequins using a user mapping file",
		Long: `List all mannequins in the organization and reattribute each one to its mapped target user.

The mapping file (--usermap) must be a YAML file as produced by 'user map'.
Each mannequin is matched to a mapping entry first by src login, then by email.
Mannequins already claimed are skipped unless --force is specified.
Entries whose dst login is empty are skipped.
Bot accounts (login ending with '[bot]') are skipped because they cannot be reclaimed.
Processing continues on per-mannequin errors; all collected errors are reported at the end.

Example:
  gh team-kit mannequin migrate --owner myorg --usermap user-map.yaml
  gh team-kit mannequin migrate --owner myorg --usermap user-map.yaml --skip-invitation --dryrun`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()

			// Load and compile usermap
			compiledMappings, err := settings.NewCompiledMappingsFromFile(mapFile)
			if err != nil {
				return fmt.Errorf("error loading mapping file: %w", err)
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

				// Bot accounts cannot be reattributed via mannequin reclamation, skip them
				if strings.Contains(targetLogin, "[bot]") {
					logger.Warn("Target user is a bot, skipping", "mannequin", mannequinLogin, "target-user", targetLogin)
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
					logger.Error("Failed to find target user, skipping", "mannequin", mannequinLogin, "target-user", targetLogin, "error", err)
					errs = append(errs, fmt.Errorf("failed to find user '%s' for mannequin '%s': %w", targetLogin, mannequinLogin, err))
					continue
				}
				if targetUser.NodeID == nil {
					logger.Error("Failed to get node ID for target user, skipping", "mannequin", mannequinLogin, "target-user", targetLogin)
					errs = append(errs, fmt.Errorf("failed to get node ID for user '%s'", targetLogin))
					continue
				}
				targetUserNodeID := targetUser.GetNodeID()

				if skipInvitation {
					if err := gh.ReattributeMannequinToUser(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
						logger.Error("Failed to reattribute mannequin, skipping", "mannequin", mannequinLogin, "target-user", targetLogin, "error", err)
						errs = append(errs, fmt.Errorf("failed to reattribute mannequin '%s': %w", mannequinLogin, err))
						continue
					}
					logger.Info("Mannequin reattributed successfully.", "mannequin", mannequinLogin, "target-user", targetLogin)
				} else {
					if err := gh.CreateAttributionInvitation(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
						logger.Error("Failed to invite user to claim mannequin, skipping", "mannequin", mannequinLogin, "target-user", targetLogin, "error", err)
						errs = append(errs, fmt.Errorf("failed to invite user to claim mannequin '%s': %w", mannequinLogin, err))
						continue
					}
					logger.Info("Attribution invitation sent.", "mannequin", mannequinLogin, "target-user", targetLogin)
				}
			}
			if len(errs) > 0 {
				return errors.Join(errs...)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Target organization ([HOST/]OWNER; uses current repository's organization if omitted)")
	f.StringVar(&mapFile, "usermap", "", "User mapping file (as produced by 'user map') for login resolution")
	f.BoolVar(&skipInvitation, "skip-invitation", false, "Skip the invitation step and directly reclaim mannequins (requires the feature to be enabled by GitHub Support)")
	f.BoolVar(&force, "force", false, "Process even mannequins that are already claimed")
	f.BoolVarP(&dryrun, "dryrun", "n", false, "Dry run: show what would be done without making changes")
	if err := cmd.MarkFlagRequired("usermap"); err != nil {
		panic(err)
	}

	return cmd
}
