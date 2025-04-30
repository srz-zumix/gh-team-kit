package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
)

func init() {
	rootCmd.AddCommand(teamListCmd)
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all teams in the organization",
	Long:  `Retrieve and display a list of all teams in the specified GitHub organization.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Println("Error: Organization name is required")
			return
		}
		org := args[0]
		client, err := gh.NewGitHubClient()
		if err != nil {
			fmt.Printf("Error creating GitHub client: %v\n", err)
			return
		}
		ctx := context.Background()
		teams, err := client.ListTeams(ctx, org)
		if err != nil {
			fmt.Printf("Error retrieving teams: %v\n", err)
			return
		}
		for _, team := range teams {
			fmt.Printf("- %s\n", *team.Name)
		}
	},
}
