package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/ddddddO/gtree"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type TreeOptions struct {
	Exporter cmdutil.Exporter
}

func init() {
	opts := &TreeOptions{}

	var owner string
	var recursive bool

	var treeCmd = &cobra.Command{
		Use:   "tree [team-slug]",
		Short: "Displays a team hierarchy in a tree structure",
		Long:  `Displays a team hierarchy in a tree structure based on the team's slug.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			var team gh.Team
			var root *gtree.Node
			if len(args) > 0 {
				teamSlug := args[0]
				team, err = gh.TeamByName(ctx, client, repository, teamSlug, false, recursive)
				if err != nil {
					return fmt.Errorf("error retrieving teams: %w", err)
				}
			} else {
				team, err = gh.TeamByOwner(ctx, client, repository, recursive)
				if err != nil {
					return fmt.Errorf("error retrieving teams: %w", err)
				}
				root = gtree.NewRoot(repository.Owner)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, team); err != nil {
					return fmt.Errorf("error exporting teams: %w", err)
				}
				return nil
			}

			if err := gtree.OutputFromRoot(client.IO.Out, GTree(root, team)); err != nil {
				return fmt.Errorf("error outputting tree: %w", err)
			}
			return nil
		},
	}

	treeCmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the team")
	treeCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Retrieve teams recursively")
	cmdutil.AddFormatFlags(treeCmd, &opts.Exporter)

	rootCmd.AddCommand(treeCmd)
}

func GTree(node *gtree.Node, team gh.Team) *gtree.Node {
	root := node
	if team.Team != nil {
		if node == nil {
			node = gtree.NewRoot(team.Team.GetSlug())
			root = node
		} else {
			node = node.Add(team.Team.GetSlug())
		}
	} else {
		if node == nil {
			return nil
		}
	}
	for _, child := range team.Child {
		GTree(node, child)
	}
	return root
}
