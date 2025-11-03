package config

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v73/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/gh/client"
	"gopkg.in/yaml.v3"
)

type Exporter struct {
	ctx    context.Context
	client *client.GitHubClient
	Owner  repository.Repository
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

func extractUserLogins(users []*github.User) []string {
	logins := make([]string, 0, len(users))
	for _, user := range users {
		logins = append(logins, *user.Login)
	}
	return logins
}

func (e *Exporter) Export() (*OrganizationConfig, error) {
	teams, err := gh.ListTeams(e.ctx, e.client, e.Owner)
	if err != nil {
		return nil, fmt.Errorf("error retrieving teams: %w", err)
	}

	teamConfigs := make([]TeamConfig, 0, len(teams))
	childTeams := make(map[string]*TeamHierarchy)
	teamHierarchy := []*TeamHierarchy{}

	for _, team := range teams {
		members, err := gh.ListTeamMembers(e.ctx, e.client, e.Owner, *team.Slug, []string{gh.TeamMembershipRoleMember}, false)
		if err != nil {
			return nil, fmt.Errorf("error retrieving team members for team %s: %w", *team.Slug, err)
		}
		maintainers, err := gh.ListTeamMembers(e.ctx, e.client, e.Owner, *team.Slug, []string{gh.TeamMembershipRoleMaintainer}, false)
		if err != nil {
			return nil, fmt.Errorf("error retrieving team maintainers for team %s: %w", *team.Slug, err)
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

		teamConfig := TeamConfig{
			Name:                *team.Name,
			Slug:                slug,
			Description:         *team.Description,
			Privacy:             *team.Privacy,
			ParentTeam:          parentSlug,
			NotificationSetting: *team.NotificationSetting,
			Maintainers:         extractUserLogins(maintainers),
			Members:             extractUserLogins(members),
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

func (e *Exporter) WriteFile(organizationConfig *OrganizationConfig, output string) (err error) {
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer func() {
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		} else if closeErr != nil {
			err = fmt.Errorf("write error: %w; close error: %v", err, closeErr)
		}
	}()
	return e.Write(organizationConfig, f)
}

func (e *Exporter) Write(organizationConfig *OrganizationConfig, w io.Writer) (err error) {
	encoder := yaml.NewEncoder(w)
	defer func() {
		closeErr := encoder.Close()
		if err == nil {
			err = closeErr
		} else if closeErr != nil {
			err = fmt.Errorf("write error: %w; encoder close error: %v", err, closeErr)
		}
	}()
	return encoder.Encode(organizationConfig)
}
