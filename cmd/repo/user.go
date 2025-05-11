package repo

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/repo/user"
)

func NewUserCmd() *cobra.Command {
	var userCmd = &cobra.Command{
		Use:   "user",
		Short: "Manage repository users",
		Long:  `Manage repository users in the organization.`,
	}

	userCmd.AddCommand(user.NewCheckCmd())
	userCmd.AddCommand(user.NewListCmd())
	userCmd.AddCommand(user.NewRemoveCmd())

	return userCmd
}
