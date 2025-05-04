package gh

import (
	"strings"

	"github.com/google/go-github/v71/github"
)

// FilterRepositoriesByNames filters a list of repositories by their full names (owner/repo).
// If the names do not include the owner, the owner is prepended.
func FilterRepositoriesByNames(repos []*github.Repository, names []string, owner string) []*github.Repository {
	nameSet := make(map[string]struct{})
	for _, name := range names {
		if !strings.Contains(name, "/") {
			name = owner + "/" + name
		}
		nameSet[name] = struct{}{}
	}

	var filteredRepos []*github.Repository
	for _, repo := range repos {
		repoFullName := repo.GetFullName() // owner/repo
		if _, exists := nameSet[repoFullName]; exists {
			filteredRepos = append(filteredRepos, repo)
		}
	}

	return filteredRepos
}
