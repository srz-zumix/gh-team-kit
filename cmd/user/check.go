package user

import (
	"context"
	"fmt"

	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"

	"github.com/spf13/cobra"
)

// NewCheckCmd creates a new `user check` command
func NewCheckCmd() *cobra.Command {
	var exitCode bool
	var owner string

	cmd := &cobra.Command{
		Use:   "check <username>",
		Short: "Check the role of a user in the organization",
		Long:  `Check the role of a specified user in the organization. If the user is not a member, it will return 'none'.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			username := args[0]

			if exitCode {
				cmd.SilenceUsage = true
			}

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			membership, err := gh.FindOrgMembership(ctx, client, repository, username)
			if err != nil {
				return fmt.Errorf("error checking membership: %w", err)
			}

			if membership != nil {
				fmt.Printf("%s\n", *membership.Role)
			} else {
				fmt.Printf("none\n")
				if exitCode {
					cmd.SilenceErrors = true
					return fmt.Errorf("user is not a member of the organization")
				}
			}

			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&exitCode, "exit-code", false, "Return an exit code of 1 if the user is not a member")
	f.StringVar(&owner, "owner", "", "Owner of the repository")

	return cmd
}
