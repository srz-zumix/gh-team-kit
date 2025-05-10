package gh

import (
	"context"
	"slices"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

// CheckTeamPermissions is a wrapper function to check team permissions for a repository.
func CheckTeamPermissions(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) (*github.Repository, error) {
	if teamSlug == "" {
		return nil, nil
	}
	return g.CheckTeamPermissions(ctx, repo.Owner, teamSlug, repo.Owner, repo.Name)
}

func ListTeamRepos(ctx context.Context, g *GitHubClient, repo repository.Repository, teamName string, roles []string, inherit bool) ([]*github.Repository, error) {
	if teamName == "" {
		return nil, nil
	}
	repos, err := g.ListTeamRepos(ctx, repo.Owner, teamName)
	if err != nil {
		return nil, err
	}

	if !inherit {
		var noInheritRepos []*github.Repository
		team, err := g.GetTeamBySlug(ctx, repo.Owner, teamName)
		if err != nil {
			return nil, err
		}
		if team != nil && team.Parent != nil {
			parentRepos, err := g.ListTeamRepos(ctx, repo.Owner, *team.Parent.Slug)
			if err != nil {
				return nil, err
			}
			for _, repo := range repos {
				d := FindRepository(repo, parentRepos)
				if CompareRepository(repo, d) != nil {
					noInheritRepos = append(noInheritRepos, repo)
				} else {
					teams, err := g.ListRepositoryTeams(ctx, *repo.Owner.Login, *repo.Name)
					if err != nil {
						return nil, err
					}
					if slices.ContainsFunc(teams, func(t *github.Team) bool {
						return *t.Slug == teamName
					}) {
						noInheritRepos = append(noInheritRepos, repo)
					}
				}
			}
			repos = noInheritRepos
		}
	}

	if len(roles) > 0 {
		var filteredRepos []*github.Repository
		for _, r := range repos {
			for _, role := range roles {
				if r.Permissions[role] {
					filteredRepos = append(filteredRepos, r)
					break
				}
			}
		}
		return filteredRepos, nil
	}
	return repos, nil
}

// ListUserRepositories is a wrapper function to retrieve all repositories associated with a specific user.
func ListUserRepositories(ctx context.Context, g *GitHubClient, username string, types []string) ([]*github.Repository, error) {
	if username == "" {
		return nil, nil
	}
	return g.ListUserRepositories(ctx, username, GetUserRepositoryTypeFilter(types))
}
