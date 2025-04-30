package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
)

func init() {
	teamListCmd.Flags().StringP("repo", "R", "", "Specify a repository to filter teams")
	rootCmd.AddCommand(teamListCmd)
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all teams in the organization",
	Long:  `Retrieve and display a list of all teams in the specified GitHub organization.`,
	Run: func(cmd *cobra.Command, args []string) {
		repoOption, _ := cmd.Flags().GetString("repo")

		repository, err := gh.ParseRepository(repoOption)
		if err != nil {
			fmt.Printf("Failed to parse repository: %v\n", err)
			return
		}

		client, err := gh.NewGitHubClientWithRepo(repository)
		if err != nil {
			fmt.Printf("Error creating GitHub client: %v\n", err)
			return
		}
		ctx := context.Background()

		owner := repository.Owner
		repo := repository.Name

		if repo != "" {
			fmt.Printf("Filtering teams for repository: %s\n", repo)
			teams, err := client.ListTeamsByRepo(ctx, owner, repo)
			if err != nil {
				fmt.Printf("Error retrieving teams for repository: %v\n", err)
				return
			}
			for _, team := range teams {
				fmt.Printf("- %s\n", *team.Name)
			}
		} else {
			teams, err := client.ListTeams(ctx, owner)
			if err != nil {
				fmt.Printf("Error retrieving teams: %v\n", err)
				return
			}
			for _, team := range teams {
				fmt.Printf("- %s\n", *team.Name)
			}
		}
	},
}
