package gh

import (
	"slices"

	"github.com/google/go-github/v71/github"
)

var PermissionsList = []string{
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

var RepoSearchTypeList = []string{
	"public",
	"internal",
	"private",
	"forks",
	"sources",
	"member",
}

var RepoVisibilityList = []string{
	"public",
	"private",
	"internal",
}

var CollaboratorAffiliationList = []string{
	"outside",
	"direct",
}

type RespositorySearchOptions struct {
	Visibility []string
	Fork       *bool
	Archived   *bool
	Mirror     *bool
	Template   *bool
}

func (opt *RespositorySearchOptions) SetFork(fork bool) {
	opt.Fork = new(bool)
	*opt.Fork = fork
}
func (opt *RespositorySearchOptions) SetArchived(archived bool) {
	opt.Archived = new(bool)
	*opt.Archived = archived
}
func (opt *RespositorySearchOptions) SetMirror(mirror bool) {
	opt.Mirror = new(bool)
	*opt.Mirror = mirror
}
func (opt *RespositorySearchOptions) SetTemplate(template bool) {
	opt.Template = new(bool)
	*opt.Template = template
}
func (opt *RespositorySearchOptions) SetSources() {
	opt.SetFork(false)
	opt.SetArchived(false)
	opt.SetMirror(false)
}

func (opt *RespositorySearchOptions) Sources() bool {
	if opt.Fork == nil || *opt.Fork {
		return false
	}
	if opt.Archived == nil || *opt.Archived {
		return false
	}
	if opt.Mirror == nil || *opt.Mirror {
		return false
	}
	return true
}

func (opt *RespositorySearchOptions) GetFilterString() string {
	if opt != nil {
		matched := 0
		for _, role := range opt.Visibility {
			if slices.Contains(RepoSearchTypeList, role) {
				matched++
			}
		}
		if matched == 1 && len(opt.Visibility) == 1 {
			return opt.Visibility[0]
		}

		if opt.Sources() {
			return "sources"
		}
		if opt.Fork != nil && *opt.Fork {
			return "forks"
		}
	}
	return "all"
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
	for _, permission := range PermissionsList {
		if permissions[permission] {
			return permission
		}
	}
	return "none"
}

func GetTeamMembershipFilter(roles []string) string {
	if roles != nil {
		matched := 0
		for _, role := range roles {
			if slices.Contains(TeamMembershipList, role) {
				matched++
			}
		}
		if matched == 1 && len(roles) == 1 {
			return roles[0]
		}
	}
	return "all"
}

func GetOrgMembershipFilter(roles []string) string {
	if roles != nil {
		matched := 0
		for _, role := range roles {
			if slices.Contains(OrgMembershipList, role) {
				matched++
			}
		}
		if matched == 1 && len(roles) == 1 {
			return roles[0]
		}
	}
	return "all"
}

func GetCollaboratorAffiliationsFilter(affiliations []string) string {
	if affiliations != nil {
		matched := 0
		for _, role := range affiliations {
			if slices.Contains(CollaboratorAffiliationList, role) {
				matched++
			}
		}
		if matched == 1 && len(affiliations) == 1 {
			return affiliations[0]
		}
	}
	return "all"
}
