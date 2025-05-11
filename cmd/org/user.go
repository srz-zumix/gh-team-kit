package org

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/org/user"
)

func NewUserCmd() *cobra.Command {
	var userCmd = &cobra.Command{
		Use:   "user",
		Short: "Manage organization role users",
		Long:  `Manage users assigned to specific roles within the organization.`}

	userCmd.AddCommand(user.NewListCmd())

	return userCmd
}
