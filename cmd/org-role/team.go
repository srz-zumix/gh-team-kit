package orgrole

import (
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/cmd/org-role/team"
)

func NewTeamCmd() *cobra.Command {
	var teamCmd = &cobra.Command{
		Use:   "team",
		Short: "Manage organization teams",
		Long:  `Manage teams within the organization.`,
	}

	teamCmd.AddCommand(team.NewAddCmd())
	teamCmd.AddCommand(team.NewListCmd())
	teamCmd.AddCommand(team.NewRemoveCmd())

	return teamCmd
}
