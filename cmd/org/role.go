package org

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/org/role"
)

func NewRoleCmd() *cobra.Command {
	var roleCmd = &cobra.Command{
		Use:   "role",
		Short: "Manage organization roles",
		Long:  `Manage roles within the organization.`,
	}

	roleCmd.AddCommand(role.NewListCmd())

	return roleCmd
}
