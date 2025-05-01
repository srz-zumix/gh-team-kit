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

	var treeCmd = &cobra.Command{
		Use:   "tree",
		Short: "Displays a team hierarchy in a tree structure",
		Long:  `Displays a team hierarchy in a tree structure based on the team's slug.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			owner, _ := cmd.Flags().GetString("owner")
			recursive, _ := cmd.Flags().GetBool("recursive")
			teamSlug := args[0]
			if teamSlug == "" {
				fmt.Println("Error: Team slug is required")
				return
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				fmt.Printf("Error parsing repository: %v\n", err)
				return
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				fmt.Printf("Error creating GitHub client: %v\n", err)
				return
			}

			team, err := gh.TeamByName(ctx, client, repository, teamSlug, false, recursive)
			if err != nil {
				fmt.Printf("Error retrieving teams: %v\n", err)
				return
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, team); err != nil {
					fmt.Printf("Error exporting teams: %v\n", err)
					return
				}
				return
			}

			if err := gtree.OutputFromRoot(client.IO.Out, GTree(nil, team)); err != nil {
				fmt.Printf("Error outputting tree: %v\n", err)
				return
			}
		},
	}

	treeCmd.Flags().StringP("owner", "", "", "The owner of the team")
	treeCmd.Flags().BoolP("recursive", "r", false, "Retrieve teams recursively")
	cmdutil.AddFormatFlags(treeCmd, &opts.Exporter)

	rootCmd.AddCommand(treeCmd)
}

func GTree(node *gtree.Node, team gh.Team) *gtree.Node {
	root := node
	if team.Team == nil {
		return root
	}
	if node == nil {
		node = gtree.NewRoot(team.Team.GetSlug())
		root = node
	} else {
		node = node.Add(team.Team.GetSlug())
	}
	for _, child := range team.Child {
		GTree(node, child)
	}
	return root
}
