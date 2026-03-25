package config

import (
	"fmt"
	"slices"
)

// validTeamPrivacyValues contains the accepted values for TeamConfig.Privacy.
var validTeamPrivacyValues = []string{"secret", "closed"}

// validTeamNotificationSettings contains the accepted values for TeamConfig.NotificationSetting.
var validTeamNotificationSettings = []string{"notifications_enabled", "notifications_disabled"}

// validTeamRepositoryPermissions contains the accepted values for TeamRepositoryPermission.Permission.
var validTeamRepositoryPermissions = []string{"admin", "maintain", "push", "triage", "pull"}

// Verify inspects the OrganizationConfig for configuration problems without making
// any API calls. It returns a slice of errors describing every issue found.
// An empty slice means the configuration is valid.
func (c *OrganizationConfig) Verify() []error {
	var errs []error

	// Build a slug set while validating individual team entries.
	teamSlugs := make(map[string]bool, len(c.Teams))
	for _, team := range c.Teams {
		if team.Slug == "" {
			errs = append(errs, fmt.Errorf("team %q has an empty slug", team.Name))
		}
		if team.Name == "" {
			errs = append(errs, fmt.Errorf("team with slug %q has an empty name", team.Slug))
		}
		if team.Slug != "" {
			if teamSlugs[team.Slug] {
				errs = append(errs, fmt.Errorf("duplicate team slug: %q", team.Slug))
			}
			teamSlugs[team.Slug] = true
		}

		if team.Privacy != "" && !slices.Contains(validTeamPrivacyValues, team.Privacy) {
			errs = append(errs, fmt.Errorf("team %q: invalid privacy value %q, must be one of %v", team.Slug, team.Privacy, validTeamPrivacyValues))
		}

		if team.NotificationSetting != "" && !slices.Contains(validTeamNotificationSettings, team.NotificationSetting) {
			errs = append(errs, fmt.Errorf("team %q: invalid notification_setting %q, must be one of %v", team.Slug, team.NotificationSetting, validTeamNotificationSettings))
		}

		for _, repoPerm := range team.Repositories {
			if repoPerm.Name == "" {
				errs = append(errs, fmt.Errorf("team %q: repository entry has an empty name", team.Slug))
			}
			if repoPerm.Permission != "" && !slices.Contains(validTeamRepositoryPermissions, repoPerm.Permission) {
				errs = append(errs, fmt.Errorf("team %q: invalid repository permission %q for %q, must be one of %v", team.Slug, repoPerm.Permission, repoPerm.Name, validTeamRepositoryPermissions))
			}
		}
	}

	// Verify hierarchy entries and external group constraints.
	errs = append(errs, verifyHierarchy(c, c.Hierarchy, teamSlugs, 0)...)

	return errs
}

// verifyHierarchy recursively walks the hierarchy tree and validates each entry.
// depth 0 means the top-level of the hierarchy (direct children of the root).
// External groups are only valid for leaf teams at depth 0 (no child teams).
func verifyHierarchy(c *OrganizationConfig, hierarchies []*TeamHierarchy, teamSlugs map[string]bool, depth int) []error {
	var errs []error
	for _, h := range hierarchies {
		if !teamSlugs[h.Slug] {
			errs = append(errs, fmt.Errorf("hierarchy references unknown team slug %q", h.Slug))
			continue
		}

		teamConfig := c.FindTeamConfigBySlug(h.Slug)
		if teamConfig == nil {
			continue
		}

		// A leaf team is one without child teams at the top hierarchy level.
		// Only leaf teams can be connected to an external group.
		isTopLevelLeafTeam := len(h.Child) == 0 && depth == 0

		if teamConfig.Group != "" {
			if !isTopLevelLeafTeam {
				if depth == 0 {
					errs = append(errs, fmt.Errorf("team %q: cannot set external group %q because the team has child teams", h.Slug, teamConfig.Group))
				} else {
					errs = append(errs, fmt.Errorf("team %q: cannot set external group %q because the team has child or parent teams", h.Slug, teamConfig.Group))
				}
			} else if len(teamConfig.Members) > 0 || len(teamConfig.Maintainers) > 0 {
				errs = append(errs, fmt.Errorf("team %q: has both external group %q and explicit members/maintainers; members/maintainers will be ignored", h.Slug, teamConfig.Group))
			}
		}

		errs = append(errs, verifyHierarchy(c, h.Child, teamSlugs, depth+1)...)
	}
	return errs
}
