package member

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var owner string

	cmd := &cobra.Command{
		Use:   "list <team-slug>",
		Short: "List members of a team",
		Long:  `List all members of the specified team in the organization.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]
			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			members, err := gh.ListTeamMembers(ctx, client, repository, teamSlug)
			if err != nil {
				return fmt.Errorf("failed to list team members: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, members); err != nil {
					return fmt.Errorf("error exporting team members: %w", err)
				}
				return nil
			}

			for _, member := range members {
				fmt.Printf("%s\n", *member.Login)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&owner, "owner", "", "", "The owner of the team")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
