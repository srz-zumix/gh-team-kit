package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/org"
)

// NewOrgCmd creates the base command for org-related operations
func NewOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org",
		Short: "Manage organizations",
		Long:  `Commands for managing GitHub organizations.`,
	}

	cmd.AddCommand(org.NewAddCmd())
	cmd.AddCommand(org.NewListCmd())
	cmd.AddCommand(org.NewRoleCmd())
	cmd.AddCommand(org.NewUserCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewOrgCmd())
}
