package user

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewSyncCmd creates a new `repo user sync` command
func NewSyncCmd() *cobra.Command {
	var repo string
	var dstHost string

	cmd := &cobra.Command{
		Use:   "sync <dst-repository...>",
		Short: "Sync direct user permissions to multiple destination repos",
		Long:  `Sync direct user collaborator permissions from the source repository to multiple destination repositories. The destination repositories can be specified by their full name (owner/repo).`,
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			srcClient, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			ctx := cmd.Context()
			for _, dstArg := range args {
			for _, dstArg := range args {
				dstRepository, err := parser.Repository(parser.RepositoryInput(dstArg))
				if err != nil {
					return fmt.Errorf("error parsing destination repository %q: %w", dstArg, err)
				}
				if dstHost != "" {
					dstRepository.Host = dstHost
				}
				dstClient := srcClient
				if repository.Host != dstRepository.Host {
					dstClient, err = gh.NewGitHubClientWithRepo(dstRepository)
					if err != nil {
						return fmt.Errorf("error creating destination GitHub client: %w", err)
					}
				}
				}

				if err := gh.SyncRepoUserPermissions(ctx, srcClient, repository, dstClient, dstRepository); err != nil {
					return fmt.Errorf("failed to sync user permissions to %s: %w", dstArg, err)
				}
				logger.Info("User permissions synced successfully.", "from", parser.GetRepositoryFullName(repository), "to", parser.GetRepositoryFullName(dstRepository))
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format '[HOST/]OWNER/REPO'")
	f.StringVar(&dstHost, "dst-host", "", "The destination host")

	return cmd
}
