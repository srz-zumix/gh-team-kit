package mannequin

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewInviteCmd creates a new cobra.Command for inviting a user to claim a mannequin.
func NewInviteCmd() *cobra.Command {
	var owner string
	var skipInvitation bool

	cmd := &cobra.Command{
		Use:   "invite <mannequin-login> <target-user-login>",
		Short: "Invite a user to claim a mannequin",
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
			org, err := gh.GetOrg(ctx, client, repository.Owner)
			if err != nil {
				return fmt.Errorf("failed to get organization '%s': %w", repository.Owner, err)
			}
			if org.NodeID == nil {
				return fmt.Errorf("failed to get node ID for organization '%s'", repository.Owner)
			}
			orgNodeID := *org.NodeID

			// Find mannequin by login
			mannequins, err := gh.ListMannequins(ctx, client, repository, nil)
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
				return fmt.Errorf("mannequin '%s' not found in organization '%s'", mannequinLogin, repository.Owner)
			}

			// Get target user node ID
			user, err := gh.FindUser(ctx, client, targetUserLogin)
			if err != nil {
				return fmt.Errorf("failed to find user '%s': %w", targetUserLogin, err)
			}
			if user.NodeID == nil {
				return fmt.Errorf("failed to get node ID for user '%s'", targetUserLogin)
			}
			targetUserNodeID := *user.NodeID

			if skipInvitation {
				if err := gh.ReattributeMannequinToUser(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to reattribute mannequin to user: %w", err)
				}
				logger.Info("Mannequin reattributed successfully.", "mannequin", mannequinLogin, "target-user", targetUserLogin)
			} else {
				if err := gh.CreateAttributionInvitation(ctx, client, repository, orgNodeID, mannequinNodeID, targetUserNodeID); err != nil {
					return fmt.Errorf("failed to invite user to claim mannequin: %w", err)
				}
				logger.Info("Attribution invitation sent.", "mannequin", mannequinLogin, "target-user", targetUserLogin)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Organization name (uses current repository's organization if omitted)")
	f.BoolVar(&skipInvitation, "skip-invitation", false, "Skip the invitation step and directly reclaim the mannequin (requires the feature to be enabled by GitHub Support)")

	return cmd
}
