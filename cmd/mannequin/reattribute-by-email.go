package mannequin

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/settings"
)

// NewReattributeByEmailCmd creates a new cobra.Command for reattributing a mannequin by email.
// When --usermap is specified, the mannequin login and target user login are resolved
// from the mapping file using the email. Otherwise it searches for the mannequin
// and the target user by email directly within the target organization.
func NewReattributeByEmailCmd() *cobra.Command {
	var owner string
	var repo string
	var mapFile string
	var skipInvitation bool
	var force bool

	cmd := &cobra.Command{
		Use:   "reattribute-by-email <email>",
		Short: "Reattribute a mannequin by email",
		Long: `Find the mannequin (by email) and the target user (by email), then send an attribution invitation.

Without --usermap, the mannequin and target user are resolved by searching their email within the target organization.
With --usermap, the mannequin login (src) and target user login (dst) are read directly from the mapping file.

The target organization can be specified with --owner ([HOST/]OWNER) or --repo/-R.

Example:
  gh team-kit mannequin reattribute-by-email user@example.com --owner myorg
  gh team-kit mannequin reattribute-by-email user@example.com --owner github.example.com/myorg --usermap user-map.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			dstRepository, err := parser.Repository(
				parser.RepositoryInput(repo),
				parser.RepositoryOwnerWithHost(owner),
			)
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			dstClient, err := gh.NewGitHubClientWithRepo(dstRepository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()

			var mannequinLogin, targetUserLogin string

			if mapFile != "" {
				// Resolve logins from mapping file using email as key
				cm, err := settings.NewCompiledMappingsFromFile(mapFile)
				if err != nil {
					return fmt.Errorf("error loading mapping file: %w", err)
				}
				m, ok := cm.ResolveEmail(email)
				if !ok {
					return fmt.Errorf("no mapping found for email '%s' in map file '%s'", email, mapFile)
				}
				mannequinLogin = m.Src
				targetUserLogin = m.Dst
			} else {
				// Find mannequin by email in target organization
				mannequin, err := gh.FindMannequinByEmail(ctx, dstClient, dstRepository, email)
				if err != nil {
					return fmt.Errorf("failed to search mannequin by email in '%s': %w", dstRepository.Owner, err)
				}
				if mannequin == nil {
					return fmt.Errorf("no mannequin found with email '%s' in organization '%s'", email, dstRepository.Owner)
				}
				mannequinLogin = string(mannequin.Login)

				// Find target user by email on target host
				dstUser, err := gh.FindUserByEmail(ctx, dstClient, email)
				if err != nil {
					return fmt.Errorf("failed to search user by email on target host: %w", err)
				}
				if dstUser == nil {
					return fmt.Errorf("no user found with email '%s' on target host", email)
				}
				targetUserLogin = dstUser.GetLogin()
			}

			// Get organization node ID
			org, err := gh.GetOrg(ctx, dstClient, dstRepository)
			if err != nil {
				return fmt.Errorf("failed to get organization '%s': %w", dstRepository.Owner, err)
			}
			if org.NodeID == nil {
				return fmt.Errorf("failed to get node ID for organization '%s'", dstRepository.Owner)
			}
			orgNodeID := *org.NodeID

			// Find mannequin by login in the target organization
			mannequin, err := gh.FindMannequinByLogin(ctx, dstClient, dstRepository, mannequinLogin)
			if err != nil {
				return fmt.Errorf("failed to find mannequin: %w", err)
			}
			if mannequin == nil {
				return fmt.Errorf("mannequin '%s' not found in organization '%s'", mannequinLogin, dstRepository.Owner)
			}
			mannequinNodeID := mannequin.NodeID()

			// Check if the mannequin is already claimed
			if !force && string(mannequin.Claimant.Login) != "" {
				return fmt.Errorf("mannequin '%s' is already claimed by '%s'; use --force to override", mannequinLogin, mannequin.Claimant.Login)
			}

			// Get target user node ID
			targetUser, err := gh.FindUser(ctx, dstClient, targetUserLogin)
			if err != nil {
				return fmt.Errorf("failed to find user '%s' on target host: %w", targetUserLogin, err)
			}
			if targetUser.NodeID == nil {
				return fmt.Errorf("failed to get node ID for user '%s'", targetUserLogin)
			}
			targetUserNodeID := targetUser.GetNodeID()

			if skipInvitation {
				if err := gh.ReattributeMannequinToUser(ctx, dstClient, dstRepository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to reattribute mannequin to user (mannequin-node-id=%s): %w", mannequinNodeID, err)
				}
				logger.Info("Mannequin reattributed successfully.", "mannequin", mannequinLogin, "mannequin-node-id", mannequinNodeID, "target-user", targetUserLogin)
			} else {
				if err := gh.CreateAttributionInvitation(ctx, dstClient, dstRepository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to invite user to claim mannequin (mannequin-node-id=%s): %w", mannequinNodeID, err)
				}
				logger.Info("Attribution invitation sent.", "mannequin", mannequinLogin, "mannequin-node-id", mannequinNodeID, "target-user", targetUserLogin)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "Target repository in the format '[HOST/]OWNER/REPO'")
	f.StringVar(&owner, "owner", "", "Target organization ([HOST/]OWNER; uses current repository's organization if omitted)")
	f.StringVar(&mapFile, "usermap", "", "User mapping file (as produced by 'user map') for login resolution")
	f.BoolVar(&skipInvitation, "skip-invitation", false, "Skip the invitation step and directly reclaim the mannequin (requires the feature to be enabled by GitHub Support)")
	f.BoolVar(&force, "force", false, "Skip the claimant check and proceed even if the mannequin is already claimed")

	return cmd
}
