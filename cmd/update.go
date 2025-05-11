package cmd

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/gh-team-kit/gh"
	"github.com/srz-zumix/gh-team-kit/parser"
	"github.com/srz-zumix/gh-team-kit/render"
)

type UpdateOptions struct {
	Exporter cmdutil.Exporter
}

func NewUpdateCmd() *cobra.Command {
	opts := &UpdateOptions{}
	var description string
	var owner string

	// Move variable definitions to the correct scope
	var notificationValue, nameValue, privacyValue, parentValue string

	cmd := &cobra.Command{
		Use:   "update <team-slug>",
		Short: "Update an existing team",
		Long:  `Update the details of an existing team in the specified organization, such as its description or settings.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			teamSlug := args[0]

			ctx := context.Background()

			repository, err := parser.Repository(parser.RepositoryOwner(owner))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("error creating GitHub client: %w", err)
			}

			var desc *string
			if cmd.Flags().Changed("description") {
				desc = &description // Use the provided description, including empty string if explicitly set
			}

			var name *string
			if cmd.Flags().Changed("name") {
				name = &nameValue
			}
			var privacy *string
			if cmd.Flags().Changed("privacy") {
				privacy = &privacyValue
			}
			var parentTeamSlug *string
			if cmd.Flags().Changed("parent") {
				parentTeamSlug = &parentValue
			}

			var enableNotification *bool
			if cmd.Flags().Changed("notification") {
				enableNotification = new(bool)
				if notificationValue == "enabled" {
					*enableNotification = true
				} else {
					*enableNotification = false
				}
			}

			team, err := gh.UpdateTeam(ctx, client, repository, teamSlug, name, desc, privacy, enableNotification, parentTeamSlug)
			if err != nil {
				return fmt.Errorf("failed to update team: %w", err)
			}

			renderer := render.NewRenderer(opts.Exporter)
			if opts.Exporter != nil {
				renderer.RenderExportedData(team)
				return nil
			}

			fmt.Printf("Team '%s' updated successfully.\n", teamSlug)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&description, "description", "d", "", "New description for the team")
	f.StringVar(&nameValue, "name", "", "New name for the team")
	f.StringVar(&owner, "owner", "", "Specify the organization owner")
	cmdutil.StringEnumFlag(cmd, &privacyValue, "privacy", "", "closed", []string{"closed", "secret"}, "Privacy level of the team")
	f.StringVarP(&parentValue, "parent", "p", "", "Parent team slug. if empty, the team will be a top-level team")
	cmdutil.StringEnumFlag(cmd, &notificationValue, "notification", "", "enabled", []string{"enabled", "disabled"}, "Enable notifications for the team")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}

func init() {
	rootCmd.AddCommand(NewUpdateCmd())
}
