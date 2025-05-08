package gh

import (
	"strings"

	"slices"

	"github.com/google/go-github/v71/github"
)

func FindRepository(target *github.Repository, repos []*github.Repository) *github.Repository {
	for _, r := range repos {
		if *r.ID == *target.ID {
			return r
		}
	}
	return nil
}

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

// FilterTeamByNames filters a list of teams by their slugs.
func FilterTeamByNames(teams []*github.Team, slugs []string) []*github.Team {
	var filteredTeams []*github.Team
	for _, team := range teams {
		teamSlug := team.GetSlug()
		if slices.Contains(slugs, teamSlug) {
			filteredTeams = append(filteredTeams, team)
		}
	}
	return filteredTeams
}
