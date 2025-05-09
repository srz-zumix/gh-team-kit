package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/user"
)

func init() {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
		Long:  `Manage users, including listing and managing organization members. Examples include listing all users in an organization or filtering by role.`,
	}

	// Add subcommands to the user command
	userCmd.AddCommand(user.NewAddCmd())
	userCmd.AddCommand(user.NewCheckCmd())
	userCmd.AddCommand(user.NewListCmd())
	userCmd.AddCommand(user.NewRemoveCmd())

	rootCmd.AddCommand(userCmd)
}
