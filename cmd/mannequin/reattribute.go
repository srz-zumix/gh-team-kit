package mannequin

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewReattributeCmd creates a new cobra.Command for reattributing a mannequin to a user.
func NewReattributeCmd() *cobra.Command {
	var owner string
	var skipInvitation bool
	var force bool

	cmd := &cobra.Command{
		Use:   "reattribute <mannequin-login> <target-user-login>",
		Short: "Reattribute a mannequin to a user",
		Long: `Send an attribution invitation to a user to claim the specified mannequin.
The target user must be a member of the organization.
Use --skip-invitation to skip the invitation step and directly reclaim the mannequin (requires the feature to be enabled by GitHub Support).`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			mannequinLogin := args[0]
			targetUserLogin := args[1]

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			// Get organization node ID
			org, err := gh.GetOrg(ctx, client, repository)
			if err != nil {
				return fmt.Errorf("failed to get organization '%s': %w", repository.Owner, err)
			}
			if org.NodeID == nil {
				return fmt.Errorf("failed to get node ID for organization '%s'", repository.Owner)
			}
			orgNodeID := *org.NodeID

			// Find mannequin by login
			mannequin, err := gh.FindMannequinByLogin(ctx, client, repository, mannequinLogin)
			if err != nil {
				return fmt.Errorf("failed to find mannequin: %w", err)
			}
			if mannequin == nil {
				return fmt.Errorf("mannequin '%s' not found in organization '%s'", mannequinLogin, repository.Owner)
			}
			mannequinNodeID := mannequin.NodeID()

			// Check if the mannequin is already claimed
			if !force && string(mannequin.Claimant.Login) != "" {
				return fmt.Errorf("mannequin '%s' is already claimed by '%s'; use --force to override", mannequinLogin, mannequin.Claimant.Login)
			}

			// Get target user node ID
			user, err := gh.FindUser(ctx, client, targetUserLogin)
			if err != nil {
				return fmt.Errorf("failed to find user '%s': %w", targetUserLogin, err)
			}
			if user.NodeID == nil {
				return fmt.Errorf("failed to get node ID for user '%s'", targetUserLogin)
			}
			targetUserNodeID := user.GetNodeID()

			if skipInvitation {
				if err := gh.ReattributeMannequinToUser(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to reattribute mannequin to user (mannequin-node-id=%s): %w", mannequinNodeID, err)
				}
				logger.Info("Mannequin reattributed successfully.", "mannequin", mannequinLogin, "mannequin-node-id", mannequinNodeID, "target-user", targetUserLogin)
			} else {
				if err := gh.CreateAttributionInvitation(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to invite user to claim mannequin (mannequin-node-id=%s): %w", mannequinNodeID, err)
				}
				logger.Info("Attribution invitation sent.", "mannequin", mannequinLogin, "mannequin-node-id", mannequinNodeID, "target-user", targetUserLogin)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization name (uses current repository's organization if omitted)")
	f.BoolVar(&skipInvitation, "skip-invitation", false, "Skip the invitation step and directly reclaim the mannequin (requires the feature to be enabled by GitHub Support)")
	f.BoolVar(&force, "force", false, "Skip the claimant check and proceed even if the mannequin is already claimed")

	return cmd
}
