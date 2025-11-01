package cmd

import (
	"github.com/spf13/cobra"
	orgrole "github.com/srz-zumix/gh-team-kit/cmd/org-role"
)

// NewOrgCmd creates the base command for org-related operations
func NewOrgCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "org-role",
		Short: "Manage organizations roles",
		Long:  `Commands for managing GitHub organizations.`,
	}

	cmd.AddCommand(orgrole.NewListCmd())
	cmd.AddCommand(orgrole.NewTeamCmd())
	cmd.AddCommand(orgrole.NewUserCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewOrgCmd())
}
