package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/member"
)

func NewMemberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member",
		Short: "Manage team members",
		Long:  `Manage team members with various subcommands, such as adding, removing, and listing members.`,
	}

	// Add subcommands to the member command
	cmd.AddCommand(member.NewAddCmd())
	cmd.AddCommand(member.NewCheckCmd())
	cmd.AddCommand(member.NewListCmd())
	cmd.AddCommand(member.NewRemoveCmd())
	cmd.AddCommand(member.NewRoleCmd())
	cmd.AddCommand(member.NewSetsCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewMemberCmd())
}
