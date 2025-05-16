package parser

import (
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// TeamSlugWithOwner splits a team slug into organization and team name, returning a repository.Repository
func TeamSlugWithOwner(owner, teamSlug string) (repository.Repository, string) {
	parts := strings.SplitN(teamSlug, "/", 2)
	if len(parts) == 2 {
		return repository.Repository{Owner: parts[0]}, parts[1]
	}
	return repository.Repository{Owner: owner}, teamSlug
}

func RepositoryFromTeamSlugs(owner string, teamSlug string) (repository.Repository, string, error) {
	repo, team := TeamSlugWithOwner(owner, teamSlug)

	owners := []string{
		owner,
		repo.Owner,
	}
	repository, err := Repository(RepositoryOwners(owners))
	if err != nil {
		return repository, team, err
	}

	if repo.Owner == "" {
		repo.Host = repository.Host
		repo.Owner = repository.Owner
	}
	return repository, team, nil
}
