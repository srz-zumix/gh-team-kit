package codereview

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
)

func NewSetCmd() *cobra.Command {
	var owner string
	var algorithm string
	var memberCount int
	var enable bool
	var disable bool
	var notifyTeam cmdflags.MutuallyExclusiveBoolFlags
	var includeChildTeamMembers cmdflags.MutuallyExclusiveBoolFlags
	var countMembersAlreadyRequested cmdflags.MutuallyExclusiveBoolFlags
	var removeTeamRequest cmdflags.MutuallyExclusiveBoolFlags
	var excludeMembers []string

	var cmd = &cobra.Command{
		Use:   "set <team-slug>",
		Short: "Set code review settings",
		Long:  `Set code review settings for a team.`,
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

			// Build settings from flags
			settings, err := gh.GetTeamCodeReviewSettings(ctx, client, repository, teamSlug)
			if err != nil {
				return fmt.Errorf("failed to get current code review settings: %w", err)
			}

			if cmd.Flags().Changed("enable") {
				settings.Enabled = true
			} else if cmd.Flags().Changed("disable") {
				settings.Enabled = false
			}

			if algorithm != "" {
				settings.Algorithm = algorithm
			}

			if memberCount > 0 {
				settings.TeamMemberCount = memberCount
			}

			if notifyTeam.IsEnabled() {
				settings.NotifyTeam = true
			} else if notifyTeam.IsDisabled() {
				settings.NotifyTeam = false
			}
			if excludeMembers == nil {
				logger.Warn("No members specified to exclude from code review assignment. Settings will be initialized.")
			}
			if !includeChildTeamMembers.IsSet() {
				logger.Warn("Include child team members flag not set. Settings will be initialized.")
			}
			if !countMembersAlreadyRequested.IsSet() {
				logger.Warn("Count members already requested flag not set. Settings will be initialized.")
			}
			if !removeTeamRequest.IsSet() {
				logger.Warn("Remove team request flag not set. Settings will be initialized.")
			}
			settings.ExcludedTeamMembers = excludeMembers
			settings.IncludeChildTeamMembers = includeChildTeamMembers.GetValue()
			settings.CountMembersAlreadyRequested = countMembersAlreadyRequested.GetValue()
			settings.RemoveTeamRequest = removeTeamRequest.GetValue()

			// Set code review settings
			if err := gh.SetTeamCodeReviewSettings(ctx, client, repository, teamSlug, settings); err != nil {
				return fmt.Errorf("failed to set code review settings: %w", err)
			}

			logger.Info("Code review settings updated successfully.", "team-slug", teamSlug)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&owner, "owner", "", "Specify the organization name")
	f.BoolVar(&enable, "enable", false, "Enable code review assignment")
	f.BoolVar(&disable, "disable", false, "Disable code review assignment")
	cmd.MarkFlagsMutuallyExclusive("enable", "disable")

	cmdutil.StringEnumFlag(cmd, &algorithm, "algorithm", "a", "", gh.TeamCodeReviewAlgorithm, "Code review assignment algorithm")
	f.IntVarP(&memberCount, "member-count", "m", 0, "Number of team members to assign for review")
	notifyTeam.AddNoPrefixFlag(cmd, "notify-team", "Notify the entire team when a review is requested", "Disable notifying the entire team")
	includeChildTeamMembers.AddNoPrefixFlag(cmd, "include-child-team-members", "Include child team members in the review", "Exclude child team members from the review")
	countMembersAlreadyRequested.AddNoPrefixFlag(cmd, "count-members-already-requested", "Count members who have already been requested for review", "Do not count members who have already been requested for review")
	removeTeamRequest.AddNoPrefixFlag(cmd, "remove-team-request", "Remove the team from the review request", "Do not remove the team from the review request")
	f.StringSliceVarP(&excludeMembers, "exclude-members", "e", nil, "List of members to exclude from code review assignment")

	return cmd
}
