package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
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
	var privacy string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new team",
		Long:  `Create a new team in the specified organization with various options such as description, privacy, and notification settings.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

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

			team, err := gh.CreateTeam(ctx, client, repository, name, description, privacy, !disableNotification, &parentTeamSlug)
			if err != nil {
				return fmt.Errorf("failed to create team: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				renderer.RenderExportedData(team)
				return nil
			}

			logger.Info("Team created successfully.", "team-slug", team.GetSlug())
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&disableNotification, "disable-notification", false, "Disable notifications for the team")
	f.StringVarP(&description, "description", "d", "", "Description of the team")
	f.StringVar(&owner, "owner", "", "Specify the organization owner")
	f.StringVarP(&parentTeamSlug, "parent", "p", "", "Slug of the parent team")
	cmdutil.StringEnumFlag(cmd, &privacy, "privacy", "", "closed", []string{"closed", "secret"}, "Privacy level of the team")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewCreateCmd())
}
