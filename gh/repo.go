package gh

import (
	"context"
	"fmt"
	"slices"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

func GetRepository(ctx context.Context, g *GitHubClient, repo repository.Repository) (*github.Repository, error) {
	return g.GetRepository(ctx, repo.Owner, repo.Name)
}

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

	return FilterByRepositoryPermissions(repos, roles), nil
}

func ListUserAccessableRepositories(ctx context.Context, g *GitHubClient, repo repository.Repository, username string, roles []string, opt *RespositorySearchOptions) ([]*github.Repository, error) {
	loginUser, err := g.GetUser(ctx, "")
	if err != nil {
		return nil, err
	}
	if username == "" || username == "@me" {
		username = *loginUser.Login
	}

	repos, err := g.ListOrganizationRepositories(ctx, repo.Owner, opt.GetFilterString())
	if err != nil {
		return nil, err
	}

	repos = opt.Filter(repos)

	var filteredRepos []*github.Repository
	for _, r := range repos {
		permissions, err := g.GetRepositoryPermission(ctx, *r.Owner.Login, *r.Name, username)
		if err != nil {
			return nil, err
		}
		if permissions == nil {
			if username == *loginUser.Login {
				r.RoleName = github.Ptr("pull")
				r.Permissions = CreatePermissionMap([]string{"pull"})
			}
		} else {
			r.RoleName = permissions.RoleName
			r.Permissions = permissions.User.Permissions
		}

		if len(roles) == 0 || HasPermission(r, roles) {
			filteredRepos = append(filteredRepos, r)
		}
	}
	return filteredRepos, nil
}

// ListRepositoryCollaborators retrieves all collaborators for a specific repository.
func ListRepositoryCollaborators(ctx context.Context, g *GitHubClient, repo repository.Repository, affiliations []string, roles []string) ([]*github.User, error) {
	collaborators, err := g.ListRepositoryCollaborators(ctx, repo.Owner, repo.Name, GetCollaboratorAffiliationsFilter(affiliations))
	if err != nil {
		return nil, fmt.Errorf("failed to list collaborators for repository %s/%s: %w", repo.Owner, repo.Name, err)
	}

	return FilterByUserPermissions(collaborators, roles), nil
}

// AddRepositoryCollaborator is a wrapper function to add a collaborator to a repository.
func AddRepositoryCollaborator(ctx context.Context, g *GitHubClient, repo repository.Repository, username string, permission string) (*github.CollaboratorInvitation, error) {
	return g.AddRepositoryCollaborator(ctx, repo.Owner, repo.Name, username, permission)
}

// RemoveRepositoryCollaborator removes a collaborator from a specific repository.
func RemoveRepositoryCollaborator(ctx context.Context, g *GitHubClient, repo repository.Repository, username string) error {
	return g.RemoveRepositoryCollaborator(ctx, repo.Owner, repo.Name, username)
}

// GetRepositoryPermission retrieves the permission level of a user for a specific repository.
func GetRepositoryPermission(ctx context.Context, g *GitHubClient, repo repository.Repository, username string) (*github.RepositoryPermissionLevel, error) {
	return g.GetRepositoryPermission(ctx, repo.Owner, repo.Name, username)
}
