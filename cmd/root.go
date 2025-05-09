/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/version"
)

var rootCmd = &cobra.Command{
	Use:     "gh-team-kit",
	Short:   "Team-related operations extensions for GitHub CLI",
	Long:    `Team-related operations extensions for GitHub CLI`,
	Version: version.Version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
