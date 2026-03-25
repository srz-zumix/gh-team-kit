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

		// isTopLevelLeafTeam is true when the team has no child teams and is at the root hierarchy
		// level (depth == 0). External groups can only be connected to such teams.
		isTopLevelLeafTeam := len(hierarchy.Child) == 0 && depth == 0

		// Look up the existing team to determine whether a pre-existing external group needs
		// to be unset before creating or updating the team. A team with an external group
		// cannot have explicit members, so the group must be removed first when the configured
		// group has changed, is empty, or the team is no longer eligible for an external group.
		existingTeam, err := gh.FindTeamBySlug(i.ctx, i.client, i.Owner, teamConfig.Slug)
		if err != nil {
			return errorList, fmt.Errorf("error looking up team %s: %w", teamConfig.Slug, err)
		}

		didUnsetExternalGroup := false
		didSetExternalGroup := false
		if existingTeam != nil && allowExternalGroups {
			existingGroup, err := gh.FindExternalGroupByTeamSlug(i.ctx, i.client, i.Owner, teamConfig.Slug)
			if err != nil {
				return errorList, fmt.Errorf("error retrieving external group for team %s: %w", teamConfig.Slug, err)
			}
			if existingGroup != nil {
				existingGroupName := existingGroup.GetGroupName()
				// Unset when the config specifies no group, the group name has changed,
				// or the team is no longer eligible for an external group.
				if teamConfig.Group == "" || existingGroupName != teamConfig.Group || !isTopLevelLeafTeam {
					err = gh.UnsetExternalGroupForTeam(i.ctx, i.client, i.Owner, teamConfig.Slug)
					if err != nil {
						errorList = append(errorList, fmt.Errorf("error removing external group for team %s: %w", teamConfig.Slug, err))
					} else {
						didUnsetExternalGroup = true
						logger.Info("unset external group before team update", "team", teamConfig.Slug, "group", existingGroupName)
					}
				} else {
					// The team is already connected to the correct external group and remains eligible;
					// mark as done to skip redundant member-removal and SetExternalGroupForTeam API calls.
					didSetExternalGroup = true
				}
			}
		}

		// Create the team if it does not exist, otherwise update it.
		// When creating, use teamConfig.Slug as the name so that GitHub generates a slug
		// that matches the configured slug. If Name and Slug differ, follow up with an
		// UpdateTeam call to set the intended display name.
		if existingTeam != nil {
			_, err = gh.UpdateTeam(i.ctx, i.client, i.Owner, teamConfig.Slug, &teamConfig.Name, &teamConfig.Description, &teamConfig.Privacy, teamConfig.NotificationSetting, teamConfig.ParentTeam)
			if err != nil {
				return errorList, fmt.Errorf("error updating team %s: %w", teamConfig.Slug, err)
			}
		} else {
			_, err = gh.CreateTeam(i.ctx, i.client, i.Owner, teamConfig.Slug, teamConfig.Description, teamConfig.Privacy, teamConfig.NotificationSetting, teamConfig.ParentTeam)
			if err != nil {
				return errorList, fmt.Errorf("error creating team %s: %w", teamConfig.Slug, err)
			}
			// If the display name differs from the slug, update the team to set the correct name.
			if teamConfig.Name != teamConfig.Slug {
				_, err = gh.UpdateTeam(i.ctx, i.client, i.Owner, teamConfig.Slug, &teamConfig.Name, &teamConfig.Description, &teamConfig.Privacy, teamConfig.NotificationSetting, teamConfig.ParentTeam)
				if err != nil {
					return errorList, fmt.Errorf("error updating name of newly created team %s: %w", teamConfig.Slug, err)
				}
			}
		}

		// Determine whether to connect an external group for this team.
		// Log the reason when the external group cannot be applied.
		if teamConfig.Group != "" && !didSetExternalGroup {
			if !allowExternalGroups {
				logger.Warn("skipping external group: organization does not support external groups", "team", teamConfig.Slug, "group", teamConfig.Group)
				errorList = append(errorList, fmt.Errorf("cannot set external group for team %s because the organization does not support external groups", teamConfig.Slug))
			} else if !isTopLevelLeafTeam {
				if depth == 0 {
					logger.Warn("skipping external group: team has child teams", "team", teamConfig.Slug, "group", teamConfig.Group)
					errorList = append(errorList, fmt.Errorf("cannot set external group for team %s because the team has child teams", teamConfig.Slug))
				} else {
					logger.Warn("skipping external group: team has child or parent teams", "team", teamConfig.Slug, "group", teamConfig.Group)
					errorList = append(errorList, fmt.Errorf("cannot set external group for team %s because the team has child or parent teams", teamConfig.Slug))
				}
			} else {
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
						// didSetExternalGroup remains false; fall through to the member-based path as fallback.
						logger.Info("falling back to config members due to external group error", "team", teamConfig.Slug, "group", teamConfig.Group)
					} else {
						didSetExternalGroup = true
					}
				}
			}
		}

		if didSetExternalGroup {
			// Warn if both external group and explicit members/maintainers are specified,
			// since members will be ignored when connecting to an external group.
			if len(teamConfig.Members) > 0 || len(teamConfig.Maintainers) > 0 {
				logger.Warn("team has both external group and explicit members/maintainers; members/maintainers will be ignored", "team", teamConfig.Slug, "group", teamConfig.Group)
			}
		} else {
			// When not connecting an external group, ensure any existing external group connection
			// is removed before adding members, unless it was already unset during the
			// pre-creation check above. A team with an external group cannot have explicit members.
			if allowExternalGroups && !didUnsetExternalGroup {
				err = gh.UnsetExternalGroupForTeam(i.ctx, i.client, i.Owner, teamConfig.Slug)
				if err != nil {
					errorList = append(errorList, fmt.Errorf("error removing external group for team %s: %w", teamConfig.Slug, err))
				}
			}

			_, err = gh.AddTeamMembers(i.ctx, i.client, i.Owner, teamConfig.Slug, teamConfig.Members, gh.TeamMembershipRoleMember, true)
			if err != nil {
				errorList = append(errorList, fmt.Errorf("error adding members to team %s: %w", teamConfig.Slug, err))
			}
			_, err = gh.AddTeamMembers(i.ctx, i.client, i.Owner, teamConfig.Slug, teamConfig.Maintainers, gh.TeamMembershipRoleMaintainer, true)
			if err != nil {
				errorList = append(errorList, fmt.Errorf("error adding maintainers to team %s: %w", teamConfig.Slug, err))
			}

			allMembers := make([]string, len(teamConfig.Members), len(teamConfig.Members)+len(teamConfig.Maintainers))
			copy(allMembers, teamConfig.Members)
			allMembers = append(allMembers, teamConfig.Maintainers...)
			err = gh.RemoveTeamMembersOther(i.ctx, i.client, i.Owner, teamConfig.Slug, allMembers)
			if err != nil {
				errorList = append(errorList, fmt.Errorf("error removing members from team %s: %w", teamConfig.Slug, err))
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
