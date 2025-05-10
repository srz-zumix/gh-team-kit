package gh

import (
	"context"
	"fmt"
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

func ListUserAccessableRepositories(ctx context.Context, g *GitHubClient, repo repository.Repository, username string, permissions []string, opt *RespositorySearchOptions) ([]*github.Repository, error) {
	if username == "" {
		return nil, nil
	}

	repos, err := g.ListOrganizationRepositories(ctx, repo.Owner, "all")
	if err != nil {
		return nil, err
	}

	// var filteredRepos []*github.Repository
	// for _, r := range repos {
	// 	if r.RoleName == nil {
	// 	}
	// }
	return repos, nil
}

// ListRepositoryCollaborators retrieves all collaborators for a specific repository.
func ListRepositoryCollaborators(ctx context.Context, g *GitHubClient, repo repository.Repository, affiliations []string, roles []string) ([]*github.User, error) {
	collaborators, err := g.ListRepositoryCollaborators(ctx, repo.Owner, repo.Name, GetCollaboratorAffiliationsFilter(affiliations))
	if err != nil {
		return nil, fmt.Errorf("failed to list collaborators for repository %s/%s: %w", repo.Owner, repo.Name, err)
	}
	if len(roles) > 0 {
		var filteredCollaborators []*github.User
		for _, c := range collaborators {
			for _, role := range roles {
				if c.Permissions[role] {
					filteredCollaborators = append(filteredCollaborators, c)
					break
				}
			}
		}
		return filteredCollaborators, nil
	}
	return collaborators, nil
}
