package gh

import (
	"github.com/google/go-github/v71/github"
)

var TeamPermissionsList = []string{
	"admin",
	"maintain",
	"push",
	"triage",
	"pull",
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
