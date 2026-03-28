package config

import (
	"fmt"
	"slices"

	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// Verify inspects the OrganizationConfig for configuration problems without making
// any API calls. Each verification error is logged via the logger package and
// counted; warnings may be logged but do not fail verification. Returns an error
// if one or more verification errors were found, or nil when no such errors exist.
func (c *OrganizationConfig) Verify() error {
	var errCount int

	logErr := func(msg string, args ...any) {
		errCount++
		logger.Error(msg, args...)
	}

	// Build a slug set while validating individual team entries.
	teamSlugs := make(map[string]bool, len(c.Teams))
	for _, team := range c.Teams {
		if team.Slug == "" {
			logErr("team has an empty slug", "name", team.Name)
		}
		if team.Name == "" {
			logErr("team has an empty name", "slug", team.Slug)
		}
		if team.Slug != "" {
			if teamSlugs[team.Slug] {
				logErr("duplicate team slug", "slug", team.Slug)
			}
			teamSlugs[team.Slug] = true
		}

		if team.Privacy != "" && !slices.Contains(gh.TeamPrivacyList, team.Privacy) {
			logErr("invalid privacy value", "team", team.Slug, "value", team.Privacy, "allowed", gh.TeamPrivacyList)
		}

		if team.NotificationSetting != "" && !slices.Contains(gh.TeamNotificationSettingList, team.NotificationSetting) {
			logErr("invalid notification_setting", "team", team.Slug, "value", team.NotificationSetting, "allowed", gh.TeamNotificationSettingList)
		}

		for _, repoPerm := range team.Repositories {
			if repoPerm.Name == "" {
				logErr("repository entry has an empty name", "team", team.Slug)
			}
			if repoPerm.Permission != "" && !slices.Contains(gh.PermissionsList, repoPerm.Permission) {
				logErr("invalid repository permission", "team", team.Slug, "repo", repoPerm.Name, "value", repoPerm.Permission, "allowed", gh.PermissionsList)
			}
		}
	}

	// Validate that parent_team_slug references exist in the teams list.
	for _, team := range c.Teams {
		if team.ParentTeam != nil && *team.ParentTeam != "" {
			if *team.ParentTeam == team.Slug {
				logErr("parent_team_slug references self", "team", team.Slug)
			} else if !teamSlugs[*team.ParentTeam] {
				logErr("parent_team_slug references unknown team slug", "team", team.Slug, "parent", *team.ParentTeam)
			}
		}
	}

	// Verify hierarchy entries and external group constraints.
	verifyHierarchy(c, c.Hierarchy, teamSlugs, 0, logErr)

	if errCount > 0 {
		return fmt.Errorf("configuration verification failed with %d issue(s)", errCount)
	}
	return nil
}

// verifyHierarchy recursively walks the hierarchy tree and validates each entry.
// depth 0 means the top-level of the hierarchy (direct children of the root).
// External groups are only valid for root-level leaf teams (depth == 0, no child teams).
func verifyHierarchy(c *OrganizationConfig, hierarchies []*TeamHierarchy, teamSlugs map[string]bool, depth int, logErr func(string, ...any)) {
	for _, h := range hierarchies {
		if h == nil {
			logErr("hierarchy contains a null entry")
			continue
		}
		if !teamSlugs[h.Slug] {
			logErr("hierarchy references unknown team slug", "slug", h.Slug)
			continue
		}

		teamConfig := c.FindTeamConfigBySlug(h.Slug)
		if teamConfig == nil {
			continue
		}

		// isTopLevelLeafTeam is true when the team has no child teams and is at the root
		// hierarchy level (depth == 0). Only such teams can be connected to an external group.
		isTopLevelLeafTeam := len(h.Child) == 0 && depth == 0

		// A top-level hierarchy entry (depth == 0) must not have parent_team_slug set,
		// and a nested entry (depth > 0) must have parent_team_slug set.
		hasParent := teamConfig.ParentTeam != nil && *teamConfig.ParentTeam != ""
		if depth == 0 && hasParent {
			logErr("top-level hierarchy team must not have parent_team_slug", "team", h.Slug, "parent", *teamConfig.ParentTeam)
		} else if depth > 0 && !hasParent {
			logErr("nested hierarchy team must have parent_team_slug", "team", h.Slug)
		}

		if teamConfig.Group != "" {
			if !isTopLevelLeafTeam {
				if depth == 0 {
					logErr("cannot set external group: team has child teams", "team", h.Slug, "group", teamConfig.Group)
				} else {
					logErr("cannot set external group: team has child or parent teams", "team", h.Slug, "group", teamConfig.Group)
				}
			} else if len(teamConfig.Members) > 0 || len(teamConfig.Maintainers) > 0 {
				logger.Warn("team has both external group and explicit members/maintainers; members/maintainers will be ignored", "team", h.Slug, "group", teamConfig.Group)
			}
		}

		verifyHierarchy(c, h.Child, teamSlugs, depth+1, logErr)
	}
}
