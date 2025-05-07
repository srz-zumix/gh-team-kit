package repo

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type SyncOptions struct {
	Exporter cmdutil.Exporter
}

func NewSyncCmd() *cobra.Command {
	opts := &SyncOptions{}
	var repo string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sync <dst-repository...>",
		Short: "Sync teams and permissions to multiple destination repos",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			for _, dstArg := range args {
				dstRepository, err := parser.Repository(parser.RepositoryInput(dstArg))
				if err != nil {
					return fmt.Errorf("error parsing destination repository: %w", err)
				}

				if dryRun {
					fmt.Printf("[DRY RUN] Would sync teams and permissions from %s to %s\n", repo, dstArg)
					continue
				}

				if err := gh.SyncRepoTeamsAndPermissions(ctx, client, repository, dstRepository); err != nil {
					return fmt.Errorf("failed to sync teams and permissions to %s: %w", dstArg, err)
				}
				fmt.Printf("Successfully synced teams and permissions to %s\n", dstArg)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Simulate the sync operation without making any changes")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
