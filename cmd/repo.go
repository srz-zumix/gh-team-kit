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
	repoCmd.AddCommand(repo.NewAddCmd())
	repoCmd.AddCommand(repo.NewCheckCmd())
	repoCmd.AddCommand(repo.NewCopyCmd())
	repoCmd.AddCommand(repo.NewDiffCmd())
	repoCmd.AddCommand(repo.NewListCmd())
	repoCmd.AddCommand(repo.NewRemoveCmd())
	repoCmd.AddCommand(repo.NewSyncCmd())

	// Add repoCmd as a subcommand of rootCmd
	rootCmd.AddCommand(repoCmd)
}
