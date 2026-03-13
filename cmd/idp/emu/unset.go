package emu

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewUnsetCmd creates a new cobra.Command for removing the connection between an external group and a team.
func NewUnsetCmd() *cobra.Command {
	var owner string

	cmd := &cobra.Command{
		Use:   "unset <team-slug>",
		Short: "Remove the connection between an external group and a team",
		Long:  "Remove the connection between an external group and a team in the organization (Enterprise Managed Users).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[0])
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			if err := gh.UnsetExternalGroupForTeam(ctx, client, repository, teamSlug); err != nil {
				return fmt.Errorf("failed to remove external group connection from team '%s': %w", teamSlug, err)
			}

			logger.Info("External group connection removed from team successfully.", "team-slug", teamSlug)
			return nil
		},
	}

	cmd.Flags().StringVar(&owner, "owner", "", "Specify the organization name")

	return cmd
}
