package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type RenameOptions struct {
	Exporter cmdutil.Exporter
}

func NewRenameCmd() *cobra.Command {
	opts := &RenameOptions{}
	var owner string

	cmd := &cobra.Command{
		Use:   "rename <team-slug> <new-name>",
		Short: "Rename an existing team",
		Long:  `Rename an existing team in the specified organization to a new name.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			newName := args[1]

			ctx := context.Background()

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			team, err := gh.RenameTeam(ctx, client, repository, teamSlug, newName)
			if err != nil {
				return fmt.Errorf("failed to rename team: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, team); err != nil {
					return fmt.Errorf("error exporting team: %w", err)
				}
				return nil
			}

			fmt.Printf("Team '%s' renamed to '%s' successfully.\n", teamSlug, newName)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization owner")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewRenameCmd())
}
