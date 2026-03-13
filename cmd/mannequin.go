package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/mannequin"
)

func NewMannequinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mannequin",
		Short: "Manage organization mannequins",
		Long:  `Manage mannequins (placeholder accounts for unclaimed users) in the organization.`,
	}

	cmd.AddCommand(mannequin.NewListCmd())
	cmd.AddCommand(mannequin.NewInviteCmd())
	cmd.AddCommand(mannequin.NewMigrateCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewMannequinCmd())
}
