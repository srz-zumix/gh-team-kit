package emu

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// GetOptions holds format flags for the get command.
type GetOptions struct {
	Exporter cmdutil.Exporter
}

// NewGetCmd creates a new cobra.Command for getting a single external group.
func NewGetCmd() *cobra.Command {
	opts := &GetOptions{}
	var owner string
	var fields []string

	cmd := &cobra.Command{
		Use:   "get <group-name>",
		Short: "Get an external group",
		Long:  "Get details of a single external group in the organization (Enterprise Managed Users).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			groupName := args[0]

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			group, err := gh.GetExternalGroupByName(ctx, client, repository, groupName)
			if err != nil {
				return fmt.Errorf("failed to get external group '%s': %w", groupName, err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			return renderer.RenderExternalGroup(group, fields)
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	cmdutil.StringSliceEnumFlag(cmd, &fields, "field", "", nil, render.ExternalGroupDetailFieldList, "Fields to display")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
