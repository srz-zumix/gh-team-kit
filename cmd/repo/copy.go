package repo

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

type CopyOptions struct {
	Exporter cmdutil.Exporter
}

func NewCopyCmd() *cobra.Command {
	opts := &CopyOptions{}
	var force bool
	var repo string

	cmd := &cobra.Command{
		Use:   "copy <dst-repository...>",
		Short: "Copy teams and permissions to multiple destination repos",
		Long:  `Copy teams and permissions from the source repository to multiple destination repositories.`,
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

				if err := gh.CopyRepoTeamsAndPermissions(ctx, client, repository, dstRepository, force); err != nil {
					return fmt.Errorf("failed to copy teams and permissions to %s: %w", dstArg, err)
				}
				fmt.Printf("Successfully copied teams and permissions to %s\n", dstArg)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite existing permissions if they exist")
	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
