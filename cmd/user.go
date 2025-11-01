package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/user"
)

func NewUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		Long:  `Manage users in the organization.`,
	}

	// Add subcommands to the user command
	cmd.AddCommand(user.NewAddCmd())
	cmd.AddCommand(user.NewCheckCmd())
	cmd.AddCommand(user.NewHovercardCmd())
	cmd.AddCommand(user.NewListCmd())
	cmd.AddCommand(user.NewRemoveCmd())
	cmd.AddCommand(user.NewReposCmd())
	cmd.AddCommand(user.NewRoleCmd())
	cmd.AddCommand(user.NewSearchCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewUserCmd())
}
