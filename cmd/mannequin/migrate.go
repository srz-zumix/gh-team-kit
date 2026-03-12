package mannequin

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewMigrateCmd creates a new cobra.Command for migrating a user by email.
// It looks up the user by email on the source host to find the mannequin login,
// then looks up the user by email on the target host to find the target user login,
// and sends an attribution invitation.
func NewMigrateCmd() *cobra.Command {
	var owner string
	var repo string
	var srcHost string
	var skipInvitation bool

	cmd := &cobra.Command{
		Use:   "migrate <email>",
		Short: "Migrate a user by email from source host to target host",
		Long: `Find the mannequin (by email on source host) and the target user (by email on target host),
then send an attribution invitation to claim the mannequin.

The source host (--src-host) is the GitHub instance where the mannequin's login originated.
The target repository can be specified with --repo/-R; otherwise the current repository is used.

Example:
  gh team-kit mannequin migrate user@example.com --src-host github.example.com
  gh team-kit mannequin migrate user@example.com --src-host github.example.com --repo myorg/myrepo`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			// Parse destination repository
			dstRepository, err := parser.Repository(
				parser.RepositoryInput(repo),
				parser.RepositoryOwner(owner),
			)
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			// Build source repository (host only)
			srcRepository := repository.Repository{Host: srcHost}

			ctx := context.Background()

			// Create clients: srcClient for source host, dstClient for target host
			srcClient, dstClient, err := gh.NewGitHubClientWith2Repos(srcRepository, dstRepository)
			if err != nil {
				return fmt.Errorf("error creating GitHub clients: %w", err)
			}

			// Find the user on the source host by email to get the mannequin login
			srcUser, err := gh.FindUserByEmail(ctx, srcClient, email)
			if err != nil {
				return fmt.Errorf("failed to search user by email on source host '%s': %w", srcHost, err)
			}
			if srcUser == nil {
				return fmt.Errorf("no user found with email '%s' on source host '%s'", email, srcHost)
			}
			mannequinLogin := srcUser.GetLogin()

			// Find the user on the target host by email to get the target user login
			dstUser, err := gh.FindUserByEmail(ctx, dstClient, email)
			if err != nil {
				return fmt.Errorf("failed to search user by email on target host: %w", err)
			}
			if dstUser == nil {
				return fmt.Errorf("no user found with email '%s' on target host", email)
			}
			targetUserLogin := dstUser.GetLogin()

			// Get organization node ID from target host
			org, err := gh.GetOrg(ctx, dstClient, dstRepository.Owner)
			if err != nil {
				return fmt.Errorf("failed to get organization '%s': %w", dstRepository.Owner, err)
			}
			if org.NodeID == nil {
				return fmt.Errorf("failed to get node ID for organization '%s'", dstRepository.Owner)
			}
			orgNodeID := *org.NodeID

			// Find mannequin by login in the target organization
			mannequins, err := gh.ListMannequins(ctx, dstClient, dstRepository, nil)
			if err != nil {
				return fmt.Errorf("failed to list mannequins: %w", err)
			}
			var mannequinNodeID string
			for _, m := range mannequins {
				if string(m.Login) == mannequinLogin {
					mannequinNodeID = fmt.Sprintf("%v", m.ID)
					break
				}
			}
			if mannequinNodeID == "" {
				return fmt.Errorf("mannequin '%s' not found in organization '%s'", mannequinLogin, dstRepository.Owner)
			}

			// Get target user node ID
			targetUser, err := gh.FindUser(ctx, dstClient, targetUserLogin)
			if err != nil {
				return fmt.Errorf("failed to find user '%s' on target host: %w", targetUserLogin, err)
			}
			if targetUser.NodeID == nil {
				return fmt.Errorf("failed to get node ID for user '%s'", targetUserLogin)
			}
			targetUserNodeID := *targetUser.NodeID

			if skipInvitation {
				if err := gh.ReattributeMannequinToUser(ctx, dstClient, dstRepository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to reattribute mannequin to user: %w", err)
				}
				logger.Info("Mannequin reattributed successfully.", "mannequin", mannequinLogin, "target-user", targetUserLogin)
			} else {
				if err := gh.CreateAttributionInvitation(ctx, dstClient, dstRepository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to invite user to claim mannequin: %w", err)
				}
				logger.Info("Attribution invitation sent.", "mannequin", mannequinLogin, "target-user", targetUserLogin)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "Target repository in the format '[HOST/]OWNER/REPO'")
	f.StringVar(&owner, "owner", "", "Organization name (uses current repository's organization if omitted)")
	f.StringVar(&srcHost, "src-host", "", "Source GitHub host (e.g. github.example.com) where mannequins originated")
	f.BoolVar(&skipInvitation, "skip-invitation", false, "Skip the invitation step and directly reclaim the mannequin (requires the feature to be enabled by GitHub Support)")
	if err := cmd.MarkFlagRequired("src-host"); err != nil {
		panic(err)
	}

	return cmd
}
