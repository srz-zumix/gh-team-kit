package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/access"
)

func NewAccessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access",
		Short: "Manage repository access for teams",
		Long:  `Manage the repositories and permissions associated with teams.`,
	}

	cmd.AddCommand(access.NewAddCmd())
	cmd.AddCommand(access.NewCopyCmd())
	cmd.AddCommand(access.NewDiffCmd())
	cmd.AddCommand(access.NewListCmd())
	cmd.AddCommand(access.NewRemoveCmd())
	cmd.AddCommand(access.NewSyncCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewAccessCmd())
}
