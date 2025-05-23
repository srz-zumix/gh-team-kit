package user

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/user/hovercard"
)

// NewHovercardCmd creates a new `user hovercard` command
func NewHovercardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hovercard",
		Short: "Get contextual hovercard information for a user",
		Long:  `Get contextual hovercard information for a user using the GitHub API.`,
	}

	cmd.AddCommand(hovercard.NewGetCmd())
	cmd.AddCommand(hovercard.NewIssueCmd())
	cmd.AddCommand(hovercard.NewOrgCmd())
	cmd.AddCommand(hovercard.NewPrCmd())
	cmd.AddCommand(hovercard.NewRepoCmd())

	return cmd
}
