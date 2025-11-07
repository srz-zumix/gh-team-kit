package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/user"
)

func NewUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage organization users",
		Long:  `Manage organization users.`,
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
	cmd.AddCommand(user.NewTeamsCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewUserCmd())
}
