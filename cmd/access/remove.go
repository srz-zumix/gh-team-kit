package access

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewRemoveCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:     "remove <[[host/]owner/]team-slug> <repository...>",
		Short:   "Revoke a team's access to repositories",
		Long:    `Revoke the specified team's access to one or more repositories in the same organization.`,
		Aliases: []string{"rm"},
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamRepo, teamSlug, err := parser.RepositoryWithTeamSlugs(args[0], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing team: %w", err)
			}
			client, err := gh.NewGitHubClientWithRepo(teamRepo)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}
			for _, input := range args[1:] {
				repo, err := parser.RepositoryInOwner(input, teamRepo)
				if err != nil {
					return fmt.Errorf("error parsing repository %q: %w", input, err)
				}
				if err := gh.RemoveTeamRepo(cmd.Context(), client, repo, teamSlug); err != nil {
					return fmt.Errorf("failed to revoke team %s access to %s: %w", teamSlug, parser.GetRepositoryFullName(repo), err)
				}
				logger.Info("Repository access revoked.", "repository", parser.GetRepositoryFullName(repo), "team-slug", teamSlug)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")
	return cmd
}
