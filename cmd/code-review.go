package cmd

import (
	"github.com/spf13/cobra"
	codereview "github.com/srz-zumix/gh-team-kit/cmd/code-review"
)

func NewCodeReviewCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "code-review",
		Short:   "Manage code reviews",
		Long:    `Manage code reviews.`,
		Aliases: []string{"review"},
	}

	cmd.AddCommand(codereview.NewGetCmd())
	cmd.AddCommand(codereview.NewSetCmd())

	return cmd
}

func init() {
	rootCmd.AddCommand(NewCodeReviewCmd())
}
