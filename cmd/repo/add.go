package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

func NewAddCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:   "add <team-slug> <permission>",
		Short: "Add a repository to a team",
		Long:  `Add a specified repository to the specified team in the organization.`,
		Args: func(cmd *cobra.Command, args []string) error {
			err := cobra.ExactArgs(2)(cmd, args)
			if err != nil {
				return err
			}
			permission := args[1]
			for _, valid := range gh.TeamPermissionsList {
				if permission == valid {
					return nil
				}
			}
			return fmt.Errorf("invalid permission '%s', valid permissions are: {%s}", permission, strings.Join(gh.TeamPermissionsList, "|"))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			permission := args[1]
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.AddTeamRepo(ctx, client, repository, teamSlug, permission); err != nil {
				return fmt.Errorf("failed to add repository to team: %w", err)
			}

			fmt.Printf("Successfully added %s permission for repository '%s/%s' to team '%s'.\n", repository.Owner, repository.Name, permission, teamSlug)
			return nil
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")

	return cmd
}
