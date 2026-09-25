package config

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v90/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/gh/client"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

type Exporter struct {
	ctx    context.Context
	client *client.GitHubClient
	Owner  repository.Repository

	teamRepos     map[string][]*github.Repository
	repoTeamSlugs map[int64][]string
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

func NewExporter(ctx context.Context, repository repository.Repository) (*Exporter, error) {
	repository.Name = "" // Clear repository name to focus on organization level
	client, err := gh.NewGitHubClientWithRepo(repository)
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub client: %w", err)
	}
	return &Exporter{
		ctx:           ctx,
		client:        client,
		Owner:         repository,
		teamRepos:     make(map[string][]*github.Repository),
		repoTeamSlugs: make(map[int64][]string),
	}, nil
}

// listTeamRepos returns the team's repositories including inherited ones, cached per team.
func (e *Exporter) listTeamRepos(slug string) ([]*github.Repository, error) {
	if repos, ok := e.teamRepos[slug]; ok {
		return repos, nil
	}
	repos, err := gh.ListTeamRepos(e.ctx, e.client, e.Owner, slug, nil, true)
	if err != nil {
		return nil, err
	}
	e.teamRepos[slug] = repos
	return repos, nil
}

// isRepoTeam reports whether the team is listed in the repository's teams, cached per repository.
func (e *Exporter) isRepoTeam(repo *github.Repository, slug string) (bool, error) {
	slugs, ok := e.repoTeamSlugs[repo.GetID()]
	if !ok {
		teams, err := gh.ListRepositoryTeams(e.ctx, e.client, repository.Repository{
			Host:  e.Owner.Host,
			Owner: repo.GetOwner().GetLogin(),
			Name:  repo.GetName(),
		})
		if err != nil {
			return false, err
		}
		slugs = make([]string, 0, len(teams))
		for _, t := range teams {
			slugs = append(slugs, t.GetSlug())
		}
		e.repoTeamSlugs[repo.GetID()] = slugs
	}
	return slices.Contains(slugs, slug), nil
}

// listDirectTeamRepos returns the team's repositories excluding those inherited from the parent team.
func (e *Exporter) listDirectTeamRepos(team *github.Team) ([]*github.Repository, error) {
	slug := team.GetSlug()
	repos, err := e.listTeamRepos(slug)
	if err != nil || team.Parent == nil {
		return repos, err
	}
	parentRepos, err := e.listTeamRepos(team.Parent.GetSlug())
	if err != nil {
		return nil, err
	}
	parentRepoMap := make(map[int64]*github.Repository, len(parentRepos))
	for _, r := range parentRepos {
		parentRepoMap[r.GetID()] = r
	}

	var directRepos []*github.Repository
	for _, repo := range repos {
		if gh.CompareRepository(repo, parentRepoMap[repo.GetID()]) != nil {
			directRepos = append(directRepos, repo)
			continue
		}
		// Same permission as the parent: only an explicit grant makes it a direct repository.
		ok, err := e.isRepoTeam(repo, slug)
		if err != nil {
			return nil, err
		}
		if ok {
			directRepos = append(directRepos, repo)
		}
	}
	return directRepos, nil
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
			logger.Warn("skipping org roles export", "error", err)
		}
	}

	for _, team := range teams {
		members, err := gh.ListTeamMembers(e.ctx, e.client, e.Owner, *team.Slug, []string{gh.TeamMembershipRoleMember}, false)
		if err != nil {
			return nil, fmt.Errorf("error retrieving team members for team %s: %w", *team.Slug, err)
		}
		if options.GetExcludeSuspended() {
			members, err = gh.UpdateUsers(e.ctx, e.client, members)
			if err != nil {
				return nil, fmt.Errorf("error updating team members for team %s: %w", *team.Slug, err)
			}
			members = gh.ExcludeSuspendedUsers(members)
		}
		maintainers, err := gh.ListTeamMembers(e.ctx, e.client, e.Owner, *team.Slug, []string{gh.TeamMembershipRoleMaintainer}, false)
		if err != nil {
			return nil, fmt.Errorf("error retrieving team maintainers for team %s: %w", *team.Slug, err)
		}
		if options.GetExcludeSuspended() {
			maintainers, err = gh.UpdateUsers(e.ctx, e.client, maintainers)
			if err != nil {
				return nil, fmt.Errorf("error updating team maintainers for team %s: %w", *team.Slug, err)
			}
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
			repos, err := e.listDirectTeamRepos(team)
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
			slices.SortFunc(repoPermissions, func(a, b TeamRepositoryPermission) int {
				return strings.Compare(a.Name, b.Name)
			})
		}

		var groupName string
		if hasExternalGroups {
			group, err := gh.FindExternalGroupByTeamSlug(e.ctx, e.client, e.Owner, slug)
			if err != nil {
				logger.Warn("skipping external group export for team", "team", slug, "error", err)
			} else if group != nil && group.GroupName != nil {
				groupName = *group.GroupName
			}
		}

		maintainerNames := gh.GetUserNames(maintainers)
		slices.Sort(maintainerNames)
		memberNames := gh.GetUserNames(members)
		slices.Sort(memberNames)

		teamConfig := TeamConfig{
			Name:                team.GetName(),
			Slug:                slug,
			Description:         team.GetDescription(),
			Privacy:             team.GetPrivacy(),
			ParentTeam:          parentSlug,
			NotificationSetting: team.GetNotificationSetting(),
			Maintainers:         maintainerNames,
			Members:             memberNames,
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
