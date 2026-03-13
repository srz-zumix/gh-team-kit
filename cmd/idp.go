package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/idp"
)

func NewIDPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "idp",
		Short: "Manage identity provider (IDP) group connections",
		Long:  `Manage identity provider (IDP) group connections for teams in the organization.`,
	}

	cmd.AddCommand(idp.NewEmuCmd())
	cmd.AddCommand(idp.NewListCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewIDPCmd())
}
