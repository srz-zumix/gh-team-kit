package gh

import (
	"slices"

	"github.com/google/go-github/v71/github"
)

var TeamPermissionsList = []string{
	"admin",
	"maintain",
	"push",
	"triage",
	"pull",
}

var TeamMembershipList = []string{
	"member",
	"maintainer",
}

var OrgMembershipList = []string{
	"member",
	"admin",
}

func GetRepositoryPermissions(repo *github.Repository) string {
	if repo != nil {
		if repo.Permissions != nil {
			return GetPermissionName(repo.Permissions)
		}
	}
	return "none"
}

func GetPermissionName(permissions map[string]bool) string {
	for _, permission := range TeamPermissionsList {
		if permissions[permission] {
			return permission
		}
	}
	return "none"
}

func GetTeamMembershipFilter(roles []string) string {
	matched := 0
	for _, role := range roles {
		if slices.Contains(TeamMembershipList, role) {
			matched++
		}
	}
	if matched == 1 && len(roles) == 1 {
		return roles[0]
	}
	return "all"
}

func GetOrgMembershipFilter(roles []string) string {
	matched := 0
	for _, role := range roles {
		if slices.Contains(OrgMembershipList, role) {
			matched++
		}
	}
	if matched == 1 && len(roles) == 1 {
		return roles[0]
	}
	return "all"
}
