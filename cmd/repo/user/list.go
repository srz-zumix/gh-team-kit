package user

import (
	"context"
	"fmt"

	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/spf13/cobra"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/parser"
	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

type ListOptions struct {
	Exporter cmdutil.Exporter
}

func NewListCmd() *cobra.Command {
	opts := &ListOptions{}
	var details bool
	var nameOnly bool
	var roles []string
	var repo string
	var suspended cmdflags.MutuallyExclusiveBoolFlags
	var affiliations []string
	var excludeOrgAdmin bool
	var withTeam bool

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List repository collaborators",
		Long:    `List all collaborators for the specified repository. You can filter the results by affiliation and role.`,
		Aliases: []string{"ls"},
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if suspended.IsSet() {
				details = true
			}

			repository, err := parser.Repository(parser.RepositoryInput(repo))
			if err != nil {
				return fmt.Errorf("error parsing repository: %w", err)
			}

			ctx := context.Background()
			client, err := gh.NewGitHubClientWithRepo(repository)
			if err != nil {
				return fmt.Errorf("failed to create GitHub client: %w", err)
			}

			collaborators, err := gh.ListRepositoryCollaborators(ctx, client, repository, affiliations, roles)
			if err != nil {
				return fmt.Errorf("failed to list collaborators for repository %s: %w", repo, err)
			}

			renderer := render.NewRenderer(opts.Exporter)

			if details {
				collaborators, err = gh.UpdateUsers(ctx, client, collaborators)
				if err != nil {
					return fmt.Errorf("failed to update collaborators: %w", err)
				}
				if suspended.IsEnabled() {
					collaborators = gh.CollectSuspendedUsers(collaborators)
				}
				if suspended.IsDisabled() {
					collaborators = gh.ExcludeSuspendedUsers(collaborators)
				}
			}

			if excludeOrgAdmin {
				collaborators, err = gh.ExcludeOrganizationAdmins(ctx, client, repository, collaborators)
				if err != nil {
					return fmt.Errorf("failed to exclude organization admins: %w", err)
				}
			}

			if withTeam {
				collaborators, err = gh.DetectUserTeams(ctx, client, repository, collaborators)
				if err != nil {
					return fmt.Errorf("failed to detect user teams: %w", err)
				}
			}

			if nameOnly {
				renderer.RenderNames(collaborators)
				return nil
			} else {
				headers := []string{"USERNAME", "ROLE"}
				if withTeam {
					headers = append(headers, "TEAM")
				}
				if details {
					headers = append(headers, "EMAIL", "SUSPENDED")
				}
				renderer.RenderUsers(collaborators, headers)
			}
			return nil
		},
	}

	f := cmd.Flags()
	cmdutil.StringSliceEnumFlag(cmd, &affiliations, "affiliation", "a", nil, gh.CollaboratorAffiliationList, "List of affiliations to filter users")
	f.BoolVar(&excludeOrgAdmin, "exclude-org-admin", false, "Exclude organization administrators from the list")
	f.BoolVarP(&details, "details", "d", false, "Include detailed information about members")
	f.StringVarP(&repo, "repo", "R", "", "Repository in the format 'owner/name'")
	f.BoolVar(&nameOnly, "name-only", false, "Output only collaborator names")
	cmdutil.StringSliceEnumFlag(cmd, &roles, "role", "r", nil, gh.PermissionsList, "List of permissions to filter users")
	suspended.AddNoPrefixFlag(cmd, "suspended", "Output only suspended members", "Exclude suspended members")
	f.BoolVar(&withTeam, "with-team", false, "Detect and display team for each user")
	cmdutil.AddFormatFlags(cmd, &opts.Exporter)

	return cmd
}
