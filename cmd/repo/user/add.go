package user

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type AddOptions struct {
	Exporter cmdutil.Exporter
}

// NewAddCmd creates a new `repo user add` command
func NewAddCmd() *cobra.Command {
	opts := &AddOptions{}
	var repo string

	cmd := &cobra.Command{
		Use:   "add <username> <permission>",
		Short: "Add a user as a collaborator to a repository",
		Long:  `Add a specified user as a collaborator to a repository with a given permission (admin, maintain, push, triage, pull).`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]
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
			invitation, err := gh.AddRepositoryCollaborator(ctx, client, repository, username, permission)
			if err != nil {
				return fmt.Errorf("failed to add user '%s' to repository: %w", username, err)
			}
			if opts.Exporter != nil {
				renderer := render.NewRenderer(opts.Exporter)
				renderer.RenderExportedData(invitation)
				return nil
			}
			fmt.Printf("Successfully added user '%s' to repository with '%s' permission\n", username, permission)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&repo, "repo", "R", "", "The repository in the format 'owner/repo'")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
