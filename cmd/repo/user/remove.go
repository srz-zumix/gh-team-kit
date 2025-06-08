package user

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewRemoveCmd creates a new `repo user remove` command
func NewRemoveCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:     "remove <username>",
		Short:   "Remove a collaborator from a repository",
		Long:    `Remove a specified user as a collaborator from the specified repository.`,
		Aliases: []string{"rm"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.RemoveRepositoryCollaborator(ctx, client, repository, username); err != nil {
				return fmt.Errorf("failed to remove collaborator from repository: %w", err)
			}

			fmt.Printf("Successfully removed user '%s' from repository '%s/%s'.\n", username, repository.Owner, repository.Name)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")

	return cmd
}
