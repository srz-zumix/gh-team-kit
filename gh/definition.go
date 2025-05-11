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
			if slices.Contains(RepoVisibilityList, role) {
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

func (opt *RespositorySearchOptions) Filter(repos []*github.Repository) []*github.Repository {
	if opt == nil {
		return repos
	}
	var filteredRepos []*github.Repository
	for _, repo := range repos {
		if opt.Fork != nil {
			if *opt.Fork != *repo.Fork {
				continue
			}
		}
		if opt.Archived != nil {
			if *opt.Archived != *repo.Archived {
				continue
			}
		}
		if opt.Mirror != nil {
			hasMirror := repo.MirrorURL != nil && *repo.MirrorURL != ""
			if *opt.Mirror != hasMirror {
				continue
			}
		}
		if opt.Template != nil {
			if *opt.Template != *repo.IsTemplate {
				continue
			}
		}
		if opt.Visibility != nil {
			if !slices.Contains(opt.Visibility, *repo.Visibility) {
				continue
			}
		}
		filteredRepos = append(filteredRepos, repo)
	}
	return filteredRepos
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

type PermissionInterface interface {
	GetPermissions() map[string]bool
}

func HasPermission(obj PermissionInterface, roles []string) bool {
	if obj != nil {
		permissions := obj.GetPermissions()
		for _, role := range roles {
			if permissions[role] {
				return true
			}
		}
	}
	return false
}

func FilterByRepositoryPermissions(repos []*github.Repository, permissions []string) []*github.Repository {
	if len(permissions) > 0 {
		var filtered []*github.Repository
		for _, r := range repos {
			if HasPermission(r, permissions) {
				filtered = append(filtered, r)
			}
		}
		return filtered
	}
	return repos
}

func FilterByUserPermissions(repos []*github.User, permissions []string) []*github.User {
	if len(permissions) > 0 {
		var filtered []*github.User
		for _, r := range repos {
			if HasPermission(r, permissions) {
				filtered = append(filtered, r)
			}
		}
		return filtered
	}
	return repos
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
