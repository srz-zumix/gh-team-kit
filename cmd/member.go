package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/member"
)

func init() {
	memberCmd := &cobra.Command{
		Use:   "member",
		Short: "Manage team members",
		Long:  `Manage team members, including adding, removing, and listing members of a team.`,
	}

	// Add subcommands to the member command
	memberCmd.AddCommand(member.NewAddCmd())
	memberCmd.AddCommand(member.NewCheckCmd())
	memberCmd.AddCommand(member.NewListCmd())
	memberCmd.AddCommand(member.NewRemoveCmd())

	rootCmd.AddCommand(memberCmd)
}
