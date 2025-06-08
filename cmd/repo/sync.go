package repo

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

type SyncOptions struct {
	Exporter cmdutil.Exporter
}

func NewSyncCmd() *cobra.Command {
	opts := &SyncOptions{}
	var repo string

	cmd := &cobra.Command{
		Use:   "sync <dst-repository...>",
		Short: "Sync teams and permissions to multiple destination repos",
		Long:  `Sync teams and permissions from the source repository to multiple destination repositories. The destination repositories can be specified by their full name (owner/repo) or just the repo name if the owner is provided as a flag.`,
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
				if repository.Host != dstRepository.Host {
					return fmt.Errorf("source and destination repositories must be on the same host: %s vs %s", repository.Host, dstRepository.Host)
				}

				if err := gh.SyncRepoTeamsAndPermissions(ctx, client, repository, dstRepository); err != nil {
					return fmt.Errorf("failed to sync teams and permissions to %s: %w", dstArg, err)
				}
				fmt.Printf("Successfully synced teams and permissions to %s\n", dstArg)
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
