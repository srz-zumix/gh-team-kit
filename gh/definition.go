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

func GetRepositoryPermissions(repo *github.Repository) string {
	if repo != nil {
		if repo.Permissions != nil {
			for _, permission := range TeamPermissionsList {
				if repo.Permissions[permission] {
					return permission
				}
			}
		}
	}
	return "none"
}

func GetMembershipFilter(roles []string) string {
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
