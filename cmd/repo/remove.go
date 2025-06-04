package repo

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewRemoveCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:     "remove <team-slug>",
		Short:   "Remove a repository from a team",
		Long:    `Remove a specified repository from the specified team in the organization.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.RemoveTeamRepo(ctx, client, repository, teamSlug); err != nil {
				return fmt.Errorf("failed to remove repository from team: %w", err)
			}

			fmt.Printf("Successfully removed repository '%s/%s' from team '%s'.\n", repository.Owner, repository.Name, teamSlug)
			return nil
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")

	return cmd
}
