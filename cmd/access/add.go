package access

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewAddCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "add <[[host/]owner/]team-slug> <permission> <repository...>",
		Short: "Grant a team access to repositories",
		Long:  `Grant the specified team a permission on one or more repositories in the same organization.`,
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamRepo, teamSlug, err := parser.RepositoryWithTeamSlugs(args[0], parser.RepositoryOwnerWithHost(owner))
			if err != nil {
				return fmt.Errorf("error parsing team: %w", err)
			}
			permission := args[1]
			if !slices.Contains(gh.PermissionsList, permission) {
				return fmt.Errorf("invalid permission %q, valid permissions are: {%s}", permission, strings.Join(gh.PermissionsList, "|"))
			}
			client, err := gh.NewGitHubClientWithRepo(teamRepo)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}
			for _, input := range args[2:] {
				repo, err := parser.RepositoryInOwner(input, teamRepo)
				if err != nil {
					return fmt.Errorf("error parsing repository %q: %w", input, err)
				}
				if err := gh.AddTeamRepo(cmd.Context(), client, repo, teamSlug, permission); err != nil {
					return fmt.Errorf("failed to grant team %s access to %s: %w", teamSlug, parser.GetRepositoryFullName(repo), err)
				}
				logger.Info("Repository access granted.", "repository", parser.GetRepositoryFullName(repo), "team-slug", teamSlug, "permission", permission)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&owner, "owner", "", "Default organization ([HOST/]OWNER)")
	return cmd
}
