package config

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/gh/client"
)

type Exporter struct {
	ctx    context.Context
	client *client.GitHubClient
	Owner  repository.Repository
}

// teamOrgRoleNames extracts role names from a TeamOrgRoleEntry. Returns nil when entry is nil.
func teamOrgRoleNames(entry *gh.TeamOrgRoleEntry) []string {
	if entry == nil {
		return nil
	}
	names := make([]string, 0, len(entry.Roles))
	for _, r := range entry.Roles {
		names = append(names, r.GetName())
	}
	return names
}

type ExportOptions struct {
	IsExportRepositories bool
	IsExportGroup        bool
	IsExportOrgRoles     bool
	ExcludeSuspended     bool
}

func (opt *ExportOptions) GetIsExportRepositories() bool {
	if opt == nil {
		return true
	}
	return opt.IsExportRepositories
}

func (opt *ExportOptions) GetIsExportGroup() bool {
	if opt == nil {
		return true
	}
	return opt.IsExportGroup
}

func (opt *ExportOptions) GetIsExportOrgRoles() bool {
	if opt == nil {
		return true
	}
	return opt.IsExportOrgRoles
}

func (opt *ExportOptions) GetExcludeSuspended() bool {
	if opt == nil {
		return false
	}
	return opt.ExcludeSuspended
}

func NewExporter(repository repository.Repository) (*Exporter, error) {
	repository.Name = "" // Clear repository name to focus on organization level
	ctx := context.Background()
	client, err := gh.NewGitHubClientWithRepo(repository)
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub client: %w", err)
	}
	return &Exporter{
		ctx:    ctx,
		client: client,
		Owner:  repository,
	}, nil
}

func (e *Exporter) Export(options *ExportOptions) (*OrganizationConfig, error) {
	teams, err := gh.ListTeams(e.ctx, e.client, e.Owner)
	if err != nil {
		return nil, fmt.Errorf("error retrieving teams: %w", err)
	}

	teamConfigs := make([]TeamConfig, 0, len(teams))
	childTeams := make(map[string]*TeamHierarchy)
	teamHierarchy := []*TeamHierarchy{}

	hasExternalGroups := false
	if options.GetIsExportGroup() {
		hasExternalGroups, err = gh.HasExternalGroupsInOrganization(e.ctx, e.client, e.Owner)
		if err != nil {
			return nil, fmt.Errorf("error checking if organization has external groups: %w", err)
		}
	}

	// Build a map from team slug to assigned org role names (user-defined roles only).
	var teamOrgRoleMap map[string]*gh.TeamOrgRoleEntry
	if options.GetIsExportOrgRoles() {
		teamOrgRoleMap, err = gh.BuildTeamOrgRoleMap(e.ctx, e.client, e.Owner)
		if err != nil {
			return nil, fmt.Errorf("error retrieving team org roles: %w", err)
		}
	}

	for _, team := range teams {
		members, err := gh.ListTeamMembers(e.ctx, e.client, e.Owner, *team.Slug, []string{gh.TeamMembershipRoleMember}, false)
		if err != nil {
			return nil, fmt.Errorf("error retrieving team members for team %s: %w", *team.Slug, err)
		}
		if options.GetExcludeSuspended() {
			members = gh.ExcludeSuspendedUsers(members)
		}
		maintainers, err := gh.ListTeamMembers(e.ctx, e.client, e.Owner, *team.Slug, []string{gh.TeamMembershipRoleMaintainer}, false)
		if err != nil {
			return nil, fmt.Errorf("error retrieving team maintainers for team %s: %w", *team.Slug, err)
		}
		if options.GetExcludeSuspended() {
			maintainers = gh.ExcludeSuspendedUsers(maintainers)
		}
		codeReviewSettings, err := gh.GetTeamCodeReviewSettings(e.ctx, e.client, e.Owner, *team.Slug)
		if err != nil {
			return nil, fmt.Errorf("error retrieving code review settings for team %s: %w", *team.Slug, err)
		}

		slug := *team.Slug
		if _, ok := childTeams[slug]; !ok {
			childTeams[slug] = &TeamHierarchy{
				Slug: slug,
			}
		}
		var parentSlug *string
		if team.Parent != nil {
			parentSlug = team.Parent.Slug
			if _, ok := childTeams[*parentSlug]; !ok {
				childTeams[*parentSlug] = &TeamHierarchy{
					Slug:  *parentSlug,
					Child: []*TeamHierarchy{childTeams[slug]},
				}
			} else {
				childTeams[*parentSlug].Child = append(childTeams[*parentSlug].Child, childTeams[slug])
			}
		} else {
			teamHierarchy = append(teamHierarchy, childTeams[slug])
		}

		var repoPermissions []TeamRepositoryPermission
		if options.GetIsExportRepositories() {
			repos, err := gh.ListTeamRepos(e.ctx, e.client, e.Owner, *team.Slug, nil, false)
			if err != nil {
				return nil, fmt.Errorf("error retrieving team repositories for team %s: %w", *team.Slug, err)
			}
			repoPermissions = make([]TeamRepositoryPermission, 0, len(repos))
			for _, repo := range repos {
				if repo.GetDisabled() {
					continue
				}
				repoPermissions = append(repoPermissions, TeamRepositoryPermission{
					Name:       *repo.Name,
					Permission: gh.GetRepositoryPermissions(repo),
				})
			}
		}

		var groupName string
		if hasExternalGroups {
			group, err := gh.FindExternalGroupByTeamSlug(e.ctx, e.client, e.Owner, slug)
			if err != nil {
				return nil, fmt.Errorf("error retrieving external groups for team %s: %w", slug, err)
			}
			if group != nil && group.GroupName != nil {
				groupName = *group.GroupName
			}
		}

		teamConfig := TeamConfig{
			Name:                *team.Name,
			Slug:                slug,
			Description:         *team.Description,
			Privacy:             *team.Privacy,
			ParentTeam:          parentSlug,
			NotificationSetting: *team.NotificationSetting,
			Maintainers:         gh.GetUserNames(maintainers),
			Members:             gh.GetUserNames(members),
			Group:               groupName,
			OrgRoles:            teamOrgRoleNames(teamOrgRoleMap[slug]),
			Repositories:        repoPermissions,
		}
		if codeReviewSettings != nil && codeReviewSettings.Enabled {
			teamConfig.CodeReviewSettings = &TeamCodeReviewSettings{
				Enabled:                      codeReviewSettings.Enabled,
				Algorithm:                    codeReviewSettings.Algorithm,
				TeamMemberCount:              codeReviewSettings.TeamMemberCount,
				NotifyTeam:                   codeReviewSettings.NotifyTeam,
				ExcludedTeamMembers:          codeReviewSettings.ExcludedTeamMembers,
				IncludeChildTeamMembers:      codeReviewSettings.IncludeChildTeamMembers,
				CountMembersAlreadyRequested: codeReviewSettings.CountMembersAlreadyRequested,
				RemoveTeamRequest:            codeReviewSettings.RemoveTeamRequest,
			}
		}
		teamConfigs = append(teamConfigs, teamConfig)
	}

	organizationConfig := &OrganizationConfig{
		Teams:     teamConfigs,
		Hierarchy: teamHierarchy,
	}

	return organizationConfig, nil
}
