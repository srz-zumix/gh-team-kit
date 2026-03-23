package cmd

import (
	"github.com/spf13/cobra"
	memberprivilege "github.com/srz-zumix/gh-team-kit/cmd/member-privilege"
)

// NewMemberPrivilegeCmd creates the base command for member privilege operations.
func NewMemberPrivilegeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "member-privilege",
		Short: "Manage organization member privileges",
		Long:  `Manage organization member privileges, such as repository permissions, repository creation, and team creation.`,
	}

	cmd.AddCommand(memberprivilege.NewBasePermissionsCmd())
	cmd.AddCommand(memberprivilege.NewCanCreateTeamsCmd())
	cmd.AddCommand(memberprivilege.NewCopyCmd())
	cmd.AddCommand(memberprivilege.NewGetCmd())
	cmd.AddCommand(memberprivilege.NewSetCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewMemberPrivilegeCmd())
}
