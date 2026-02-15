package member

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewCheckCmd() *cobra.Command {
	var exitCode bool
	var owner string

	cmd := &cobra.Command{
		Use:   "check <team-slug> <username>",
		Short: "Check if a user is a member of a team",
		Long:  `Check if a specified user is a member of the specified team in the organization.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[1]
			if exitCode {
				cmd.SilenceUsage = true
			}

			repository, teamSlug, err := parser.RepositoryFromTeamSlugs(owner, args[0])
			if err != nil {
				return fmt.Errorf("error parsing repository with team slug: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			membership, err := gh.FindTeamMembership(ctx, client, repository, teamSlug, username)
			if err != nil {
				return fmt.Errorf("error checking membership: %w", err)
			}

			if membership != nil {
				fmt.Printf("%s\n", *membership.Role)
			} else {
				fmt.Printf("none\n")
				if exitCode {
					cmd.SilenceErrors = true
					return fmt.Errorf("user is not a member of the team")
				}
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&exitCode, "exit-code", false, "Return an exit code of 1 if the user is not a member")
	f.StringVar(&owner, "owner", "", "Specify the organization name")

	return cmd
}
