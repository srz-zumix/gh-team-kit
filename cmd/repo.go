package cmd

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/repo"
)

func NewRepoCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "repo",
		Short: "Manage repositories",
		Long:  `Manage repositories.`,
	}

	cmd.AddCommand(repo.NewAddCmd())
	cmd.AddCommand(repo.NewCheckCmd())
	cmd.AddCommand(repo.NewCopyCmd())
	cmd.AddCommand(repo.NewDiffCmd())
	cmd.AddCommand(repo.NewListCmd())
	cmd.AddCommand(repo.NewRemoveCmd())
	cmd.AddCommand(repo.NewSyncCmd())
	cmd.AddCommand(repo.NewUserCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewRepoCmd())
}
