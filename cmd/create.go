package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
)

type CreateOptions struct {
	Exporter cmdutil.Exporter
}

func NewCreateCmd() *cobra.Command {
	opts := &CreateOptions{}
	var description string
	var disableNotification bool
	var owner string
	var parentTeamSlug string
	var secret bool

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new team",
		Long:  `Create a new team in the specified organization with various options such as description, privacy, and notification settings.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			privacy := "closed"
			if secret {
				privacy = "secret"
			}

			ctx := context.Background()

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			// Check if the team already exists
			exists, err := gh.IsExistsTeam(ctx, client, repository, name)
			if err != nil {
				return fmt.Errorf("failed to check if team exists: %w", err)
			}
			if exists {
				return fmt.Errorf("team '%s' already exists", name)
			}

			team, err := gh.CreateTeam(ctx, client, repository, name, description, privacy, !disableNotification, parentTeamSlug)
			if err != nil {
				return fmt.Errorf("failed to create team: %w", err)
			}

			if opts.Exporter != nil {
				if err := client.Write(opts.Exporter, team); err != nil {
					return fmt.Errorf("error exporting team: %w", err)
				}
				return nil
			}

			fmt.Printf("Team '%s' created successfully.\n", team.GetName())
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&disableNotification, "disable-notification", false, "Disable notifications for the team")
	f.StringVarP(&description, "description", "d", "", "Description of the team")
	f.StringVar(&owner, "owner", "", "Specify the organization owner (optional)")
	f.StringVarP(&parentTeamSlug, "parent", "p", "", "Slug of the parent team")
	f.BoolVar(&secret, "secret", false, "Set the team as secret")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewCreateCmd())
}
