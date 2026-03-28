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

	// Build a slug set and config map while validating individual team entries.
	teamSlugs := make(map[string]bool, len(c.Teams))
	teamConfigs := make(map[string]*TeamConfig, len(c.Teams))
	for i, team := range c.Teams {
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
			teamConfigs[team.Slug] = &c.Teams[i]
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
	visited := make(map[string]bool)
	verifyHierarchy(teamConfigs, c.Hierarchy, teamSlugs, "", visited, logErr)

	if errCount > 0 {
		return fmt.Errorf("configuration verification failed with %d issue(s)", errCount)
	}
	return nil
}

// verifyHierarchy recursively walks the hierarchy tree and validates each entry.
// expectedParent is the slug of the parent team expected by the hierarchy structure;
// it is empty for top-level entries.
// visited tracks all slugs seen so far across the entire traversal to detect duplicates/cycles.
// External groups are only valid for root-level leaf teams (depth == 0, no child teams).
func verifyHierarchy(teamConfigs map[string]*TeamConfig, hierarchies []*TeamHierarchy, teamSlugs map[string]bool, expectedParent string, visited map[string]bool, logErr func(string, ...any)) {
	for _, h := range hierarchies {
		if h == nil {
			logErr("hierarchy contains a null entry")
			continue
		}
		if !teamSlugs[h.Slug] {
			logErr("hierarchy references unknown team slug", "slug", h.Slug)
			continue
		}
		if visited[h.Slug] {
			logErr("duplicate team slug in hierarchy", "slug", h.Slug)
			continue
		}
		visited[h.Slug] = true

		teamConfig := teamConfigs[h.Slug]
		if teamConfig == nil {
			logErr("hierarchy references team slug not found in teams list", "slug", h.Slug)
			continue
		}

		// isTopLevelLeafTeam is true when the team has no child teams and is at the root
		// hierarchy level (expectedParent == ""). Only such teams can be connected to an external group.
		isTopLevelLeafTeam := len(h.Child) == 0 && expectedParent == ""

		// Validate that parent_team_slug matches the hierarchy structure.
		actualParent := ""
		if teamConfig.ParentTeam != nil {
			actualParent = *teamConfig.ParentTeam
		}
		if actualParent != expectedParent {
			if expectedParent == "" {
				logErr("top-level hierarchy team must not have parent_team_slug", "team", h.Slug, "parent", actualParent)
			} else if actualParent == "" {
				logErr("nested hierarchy team must have parent_team_slug", "team", h.Slug, "expected_parent", expectedParent)
			} else {
				logErr("parent_team_slug does not match hierarchy", "team", h.Slug, "expected_parent", expectedParent, "actual_parent", actualParent)
			}
		}

		if teamConfig.Group != "" {
			if !isTopLevelLeafTeam {
				if expectedParent == "" {
					logErr("cannot set external group: team has child teams", "team", h.Slug, "group", teamConfig.Group)
				} else {
					logErr("cannot set external group: team has child or parent teams", "team", h.Slug, "group", teamConfig.Group)
				}
			} else if len(teamConfig.Members) > 0 || len(teamConfig.Maintainers) > 0 {
				logger.Warn("team has both external group and explicit members/maintainers; members/maintainers will be ignored", "team", h.Slug, "group", teamConfig.Group)
			}
		}

		verifyHierarchy(teamConfigs, h.Child, teamSlugs, h.Slug, visited, logErr)
	}
}
