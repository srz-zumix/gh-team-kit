/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/version"
	"github.com/srz-zumix/go-gh-extension/pkg/actions"
	"github.com/srz-zumix/go-gh-extension/pkg/gh/guardrails"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

var (
	logLevel string
	readOnly bool
)

var rootCmd = &cobra.Command{
	Use:     "gh-team-kit",
	Short:   "Team-related operations extensions for GitHub CLI",
	Long:    `Team-related operations extensions for GitHub CLI`,
	Version: version.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		logger.SetLogLevel(logLevel)
		guardrails.NewGuardrail(guardrails.ReadOnlyOption(readOnly))
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	if actions.IsRunsOn() {
		rootCmd.SetErrPrefix(actions.GetErrorPrefix())
	}
	logger.AddCmdFlag(rootCmd, rootCmd.PersistentFlags(), &logLevel, "log-level", "L")
	rootCmd.PersistentFlags().BoolVar(&readOnly, "read-only", false, "Run in read-only mode (prevent write operations)")
}
