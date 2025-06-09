package repo

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewAddCmd() *cobra.Command {
	var repo string

	cmd := &cobra.Command{
		Use:   "add <team-slug> <permission>",
		Short: "Add a repository to a team",
		Long:  `Add a specified repository to the specified team in the organization.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			permission := args[1]
			if !slices.Contains(gh.PermissionsList, permission) {
				return fmt.Errorf("invalid permission '%s', valid permissions are: {%s}", permission, strings.Join(gh.PermissionsList, "|"))
			}

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

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")

	return cmd
}
