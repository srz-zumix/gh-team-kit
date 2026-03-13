package idp

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/idp/emu"
)

func NewEmuCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emu",
		Short: "Manage external groups (Enterprise Managed Users)",
		Long:  `Manage external groups for Enterprise Managed Users (EMU) organizations.`,
	}

	cmd.AddCommand(emu.NewListCmd())
	cmd.AddCommand(emu.NewGetCmd())
	cmd.AddCommand(emu.NewSetCmd())
	cmd.AddCommand(emu.NewUnsetCmd())
	cmd.AddCommand(emu.NewTeamsCmd())

	return cmd
}
