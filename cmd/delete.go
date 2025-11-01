package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewDeleteCmd() *cobra.Command {
	var owner string
	var withChild bool
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <team-slug>",
		Short:   "Delete a team",
		Long:    `Delete a specified team from the organization. Ensure that the team is no longer needed as this action is irreversible.`,
		Aliases: []string{"del"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			team := args[0]

			ctx := context.Background()

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("failed to parse repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			// Find the team by slug
			teamDetails, err := gh.FindTeamBySlug(ctx, client, repository, team)
			if err != nil {
				return fmt.Errorf("failed to find team '%s': %w", team, err)
			}
			if teamDetails == nil {
				return fmt.Errorf("team '%s' does not exist", team)
			}

			// Check member count and repository count if force is not enabled
			if !force {
				if teamDetails.ReposCount != nil && *teamDetails.ReposCount > 0 {
					return fmt.Errorf("team '%s' has %d repositories. Use --force to skip this check", team, *teamDetails.ReposCount)
				}
				if teamDetails.MembersCount != nil && *teamDetails.MembersCount > 0 {
					return fmt.Errorf("team '%s' has %d members. Use --force to skip this check", team, *teamDetails.MembersCount)
				}
			}

			// Check if the team has child teams
			hasChildTeams, err := gh.HasChildTeams(ctx, client, repository, team)
			if err != nil {
				return fmt.Errorf("failed to check child teams for '%s': %w", team, err)
			}
			if hasChildTeams && !withChild {
				return fmt.Errorf("team '%s' has child teams. Use --with-child to delete", team)
			}

			if err := gh.DeleteTeam(ctx, client, repository, team); err != nil {
				return fmt.Errorf("failed to delete team '%s': %w", team, err)
			}

			fmt.Printf("Team '%s' deleted successfully\n", team)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization owner")
	f.BoolVar(&withChild, "with-child", false, "Allow deletion of a team with child teams")
	f.BoolVarP(&force, "force", "f", false, "Skip member and repository count checks")

	return cmd
}

func init() {
	rootCmd.AddCommand(NewDeleteCmd())
}
