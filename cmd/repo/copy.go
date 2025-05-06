package repo

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type CopyOptions struct {
	Exporter cmdutil.Exporter
}

func NewCopyCmd() *cobra.Command {
	opts := &CopyOptions{}
	var repo string
	var force bool

	cmd := &cobra.Command{
		Use:   "copy <dst...>",
		Short: "Copy teams and permissions to multiple destination repos",
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

				if err := gh.CopyRepoTeamsAndPermissions(ctx, client, repository, dstRepository, force); err != nil {
					return fmt.Errorf("failed to copy teams and permissions to %s: %w", dstArg, err)
				}
				fmt.Printf("Successfully copied teams and permissions to %s\n", dstArg)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force overwrite existing permissions if they exist")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
