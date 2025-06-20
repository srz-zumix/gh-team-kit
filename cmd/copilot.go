package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/copilot"
)

func NewCopilotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "copilot",
		Short: "Manage Copilot for teams",
		Long:  `Manage Copilot for teams with various subcommands.`,
	}

	cmd.AddCommand(copilot.NewMetricsCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewCopilotCmd())
}
