package user

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

// NewCopyCmd creates a new `repo user copy` command
func NewCopyCmd() *cobra.Command {
	var force bool
	var repo string
	var dstHost string

	cmd := &cobra.Command{
		Use:   "copy <dst-repository...>",
		Short: "Copy direct user permissions to multiple destination repos",
		Long:  `Copy direct user collaborator permissions from the source repository to multiple destination repositories. The destination repositories can be specified by their full name ([HOST/]OWNER/REPO).`,
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

				if err := gh.CopyRepoUserPermissions(ctx, srcClient, repository, dstClient, dstRepository, force); err != nil {
					return fmt.Errorf("failed to copy user permissions to %s: %w", parser.GetRepositoryFullName(dstRepository), err)
				}
				logger.Info("User permissions copied successfully.", "from", parser.GetRepositoryFullName(repository), "to", parser.GetRepositoryFullName(dstRepository))
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&force, "force", "f", false, "Force overwrite existing permissions if they exist")
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format '[HOST/]OWNER/REPO'")
	f.StringVar(&dstHost, "dst-host", "", "The destination host")

	return cmd
}
