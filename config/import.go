package config

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/gh/client"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
	"gopkg.in/yaml.v3"
)

type Importer struct {
	ctx    context.Context
	client *client.GitHubClient
	Owner  repository.Repository
}

func NewImporter(ctx context.Context, repository repository.Repository) (*Importer, error) {
	repository.Name = "" // Clear repository name to focus on organization level
	client, err := gh.NewGitHubClientWithRepo(repository)
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub client: %w", err)
	}
	return &Importer{
		ctx:    ctx,
		client: client,
		Owner:  repository,
	}, nil
}

func (i *Importer) importTeam(organizationConfig *OrganizationConfig, teamHierarchies []*TeamHierarchy, allowExternalGroups bool, depth int) ([]error, error) {
	errorList := []error{}

	for _, hierarchy := range teamHierarchies {
		teamConfig := organizationConfig.FindTeamConfigBySlug(hierarchy.Slug)
		if teamConfig == nil {
			return errorList, fmt.Errorf("team config not found for slug: %s", hierarchy.Slug)
		}

		_, err := gh.CreateOrUpdateTeam(i.ctx, i.client, i.Owner, teamConfig.Slug, &teamConfig.Name, teamConfig.Description, teamConfig.Privacy, teamConfig.NotificationSetting, teamConfig.ParentTeam)
		if err != nil {
			return errorList, fmt.Errorf("error creating or updating team %s: %w", teamConfig.Slug, err)
		}

		// External group handling:
		// - When teamConfig.Group is non-empty, we attempt to connect the team to the given EMU external group.
		//   * This is only allowed when the organization supports external groups (allowExternalGroups == true).
		//   * External groups are only supported for "leaf" teams. If the team has child teams or a parent team
		//     (depth > 0), the import will record an error instead of applying the external group.
		//   * A team cannot have explicit members when connected to an external group. Therefore, we skip adding
		//     members/maintainers and remove all existing members (including the caller auto-added by CreateTeam)
		//     before connecting the external group.
		// - When teamConfig.Group is empty and the organization supports external groups, we proactively remove any
		//   existing external group connection for the team. This means that omitting "group" in the import config is
		//   treated as "no external group" for that team.
		willSetExternalGroup := teamConfig.Group != "" && allowExternalGroups && len(hierarchy.Child) == 0 && depth == 0

		// Warn if both external group and explicit members/maintainers are specified,
		// since members will be ignored when connecting to an external group.
		if willSetExternalGroup && (len(teamConfig.Members) > 0 || len(teamConfig.Maintainers) > 0) {
			logger.Warn("team has both external group and explicit members/maintainers; members/maintainers will be ignored", "team", teamConfig.Slug, "group", teamConfig.Group)
		}

		if !willSetExternalGroup {
			_, err = gh.AddTeamMembers(i.ctx, i.client, i.Owner, teamConfig.Slug, teamConfig.Members, gh.TeamMembershipRoleMember, true)
			if err != nil {
				errorList = append(errorList, err)
			}

			_, err = gh.AddTeamMembers(i.ctx, i.client, i.Owner, teamConfig.Slug, teamConfig.Maintainers, gh.TeamMembershipRoleMaintainer, true)
			if err != nil {
				errorList = append(errorList, err)
			}
		}

		if teamConfig.Group != "" {
			if !allowExternalGroups {
				errorList = append(errorList, fmt.Errorf("cannot set external group for team %s because the organization does not support external groups", teamConfig.Slug))
			} else {
				if len(hierarchy.Child) == 0 && depth == 0 {
					// Remove all team members before connecting an external group.
					// CreateTeam automatically adds the calling user as a member, and a team
					// with explicit members cannot be mapped to an Identity Provider Group.
					err = gh.RemoveTeamMembersOther(i.ctx, i.client, i.Owner, teamConfig.Slug, []string{})
					if err != nil {
						errorList = append(errorList, fmt.Errorf("error removing members from team %s before setting external group: %w", teamConfig.Slug, err))
					} else {
						_, err = gh.SetExternalGroupForTeam(i.ctx, i.client, i.Owner, teamConfig.Group, teamConfig.Slug)
						if err != nil {
							errorList = append(errorList, fmt.Errorf("error setting external group '%s' for team %s: %w", teamConfig.Group, teamConfig.Slug, err))
						}
					}
				} else {
					if depth == 0 {
						errorList = append(errorList, fmt.Errorf("cannot set external group for team %s because the team has child teams", teamConfig.Slug))
					} else {
						errorList = append(errorList, fmt.Errorf("cannot set external group for team %s because the team has child or parent teams", teamConfig.Slug))
					}
				}
			}
		} else {
			if allowExternalGroups {
				// If the organization has external groups, we remove any existing group connection for teams
				// that do not have a group specified in the import config.
				err = gh.UnsetExternalGroupForTeam(i.ctx, i.client, i.Owner, teamConfig.Slug)
				if err != nil {
					errorList = append(errorList, fmt.Errorf("error removing external group for team %s: %w", teamConfig.Slug, err))
				}
			}
			allMembers := append(teamConfig.Members, teamConfig.Maintainers...)
			err = gh.RemoveTeamMembersOther(i.ctx, i.client, i.Owner, teamConfig.Slug, allMembers)
			if err != nil {
				errorList = append(errorList, err)
			}
		}

		if len(teamConfig.Repositories) > 0 {
			for _, repoPerm := range teamConfig.Repositories {
				err = gh.AddTeamRepo(i.ctx, i.client, repository.Repository{Owner: i.Owner.Owner, Name: repoPerm.Name}, teamConfig.Slug, repoPerm.Permission)
				if err != nil {
					errorList = append(errorList, fmt.Errorf("error adding repository %s to team %s: %w", repoPerm.Name, teamConfig.Slug, err))
				}
			}
		}

		// Assign org custom roles to the team.
		for _, roleName := range teamConfig.OrgRoles {
			err = gh.AssignOrgRoleToTeam(i.ctx, i.client, i.Owner, teamConfig.Slug, roleName)
			if err != nil {
				errorList = append(errorList, fmt.Errorf("error assigning org role '%s' to team %s: %w", roleName, teamConfig.Slug, err))
			}
		}

		if teamConfig.CodeReviewSettings != nil {
			err = gh.SetTeamCodeReviewSettings(i.ctx, i.client, i.Owner, teamConfig.Slug, &gh.TeamCodeReviewSettings{
				TeamSlug:                     teamConfig.Slug,
				Enabled:                      teamConfig.CodeReviewSettings.Enabled,
				Algorithm:                    teamConfig.CodeReviewSettings.Algorithm,
				NotifyTeam:                   teamConfig.CodeReviewSettings.NotifyTeam,
				ExcludedTeamMembers:          teamConfig.CodeReviewSettings.ExcludedTeamMembers,
				IncludeChildTeamMembers:      teamConfig.CodeReviewSettings.IncludeChildTeamMembers,
				CountMembersAlreadyRequested: teamConfig.CodeReviewSettings.CountMembersAlreadyRequested,
				RemoveTeamRequest:            teamConfig.CodeReviewSettings.RemoveTeamRequest,
			})
			if err != nil {
				errorList = append(errorList, fmt.Errorf("error updating code review settings for team %s: %w", teamConfig.Slug, err))
			}
		}

		childErrorList, err := i.importTeam(organizationConfig, hierarchy.Child, allowExternalGroups, depth+1)
		if err != nil {
			return errorList, err
		}
		errorList = append(errorList, childErrorList...)
	}
	return errorList, nil
}

func (i *Importer) Import(organizationConfig *OrganizationConfig) error {
	hasExternalGroups, err := gh.HasExternalGroupsInOrganization(i.ctx, i.client, i.Owner)
	if err != nil {
		return fmt.Errorf("error checking if organization has external groups: %w", err)
	}

	errorList, err := i.importTeam(organizationConfig, organizationConfig.Hierarchy, hasExternalGroups, 0)
	if err != nil {
		return err
	}
	if len(errorList) > 0 {
		return fmt.Errorf("encountered errors during import: %v", errorList)
	}
	return nil
}

func (i *Importer) ReadFile(input string) (c *OrganizationConfig, err error) {
	f, err := os.Open(input)
	if err != nil {
		return nil, fmt.Errorf("error opening input file: %w", err)
	}
	// Ensure that file close errors do not overwrite previous errors
	defer func() {
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		} else if closeErr != nil {
			err = fmt.Errorf("read error: %w; additionally, error closing file: %v", err, closeErr)
		}
	}()
	return i.Read(f)
}

func (i *Importer) Read(r io.Reader) (*OrganizationConfig, error) {
	var cfg OrganizationConfig
	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
