package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/repo"
)

func init() {
	var repoCmd = &cobra.Command{
		Use:   "repo",
		Short: "Manage team repositories",
		Long:  `Manage team repositories with various subcommands.`,
	}

	// Add subcommand of repoCmd
	repoCmd.AddCommand(repo.NewListCmd())
	repoCmd.AddCommand(repo.NewCheckCmd())

	// Add repoCmd as a subcommand of rootCmd
	rootCmd.AddCommand(repoCmd)
}
