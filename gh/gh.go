package gh

import (
	"context"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

// ListTeams is a wrapper function that uses a Repository object to call either ListTeams or ListTeamsByRepo.
func ListTeams(ctx context.Context, g *GitHubClient, repo repository.Repository) ([]*github.Team, error) {
	if repo.Name != "" {
		return g.ListTeamsByRepo(ctx, repo.Owner, repo.Name)
	}
	return g.ListTeams(ctx, repo.Owner)
}

// GetTeamByName retrieves a team by its name.
func GetTeamByName(ctx context.Context, g *GitHubClient, repo repository.Repository, teamName string) (*github.Team, error) {
	return g.GetTeamBySlug(ctx, repo.Owner, teamName)
}

// ListChildTeams is a wrapper function that calls the ListChildTeams API.
func ListChildTeams(ctx context.Context, g *GitHubClient, repo repository.Repository, parentSlug string) ([]*github.Team, error) {
	return g.ListChildTeams(ctx, repo.Owner, parentSlug)
}

// CheckTeamPermissions is a wrapper function to check team permissions for a repository.
func CheckTeamPermissions(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) (*github.Repository, error) {
	if teamSlug == "" {
		return nil, nil
	}
	return g.CheckTeamPermissions(ctx, repo.Owner, teamSlug, repo.Name)
}

// RemoveTeamRepo is a wrapper function to remove a repository from a team.
func RemoveTeamRepo(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) error {
	return g.RemoveTeamRepo(ctx, repo.Owner, teamSlug, repo.Name)
}

// AddTeamRepo is a wrapper function to add a repository to a team.
func AddTeamRepo(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, permission string) error {
	return g.AddTeamRepo(ctx, repo.Owner, teamSlug, repo.Name, permission)
}

// ListTeamMembers is a wrapper function to retrieve all members of a specific team.
func ListTeamMembers(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) ([]*github.User, error) {
	return g.ListTeamMembers(ctx, repo.Owner, teamSlug)
}

// AddTeamMember is a wrapper function to add or update a team member.
func AddTeamMember(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, username string, role string) error {
	return g.AddTeamMember(ctx, repo.Owner, teamSlug, username, role)
}

// RemoveTeamMember is a wrapper function to remove a user from a team.
func RemoveTeamMember(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, username string) error {
	return g.RemoveTeamMember(ctx, repo.Owner, teamSlug, username)
}

type Team struct {
	Team  *github.Team
	Child []Team
}

func (t *Team) Flatten() []*github.Team {
	var teams []*github.Team
	if t.Team != nil {
		teams = append(teams, t.Team)
	}
	for _, child := range t.Child {
		teams = append(teams, child.Flatten()...)
	}
	return teams
}

func TeamByOwner(ctx context.Context, g *GitHubClient, repo repository.Repository, recursive bool) (Team, error) {
	var t Team
	if repo.Owner == "" {
		return t, nil
	}
	teams, err := g.ListTeams(ctx, repo.Owner)
	if err != nil {
		return t, err
	}
	for _, team := range teams {
		if team.Slug != nil && team.Parent == nil {
			c, err := TeamByName(ctx, g, repo, *team.Slug, false, recursive)
			if err != nil {
				return t, err
			}
			t.Child = append(t.Child, c)
		}
	}
	return t, nil
}

func TeamByName(ctx context.Context, g *GitHubClient, repo repository.Repository, teamName string, child bool, recursive bool) (Team, error) {
	var t Team
	if teamName == "" {
		return t, nil
	}
	if child {
		teams, err := g.ListChildTeams(ctx, repo.Owner, teamName)
		if err != nil {
			return t, err
		}
		for _, childTeam := range teams {
			if childTeam.Slug != nil {
				if recursive {
					recursiveTeams, err := TeamByName(ctx, g, repo, *childTeam.Slug, child, recursive)
					if err != nil {
						return t, err
					}
					t.Child = append(t.Child, recursiveTeams)
				} else {
					t.Child = append(t.Child, Team{Team: childTeam})
				}
			}
		}
	} else {
		team, err := g.GetTeamBySlug(ctx, repo.Owner, teamName)
		if err != nil {
			return t, err
		}
		t.Team = team
		if recursive {
			teams, err := g.ListChildTeams(ctx, repo.Owner, teamName)
			if err != nil {
				return t, err
			}
			for _, childTeam := range teams {
				if childTeam.Slug != nil {
					recursiveTeams, err := TeamByName(ctx, g, repo, *childTeam.Slug, child, recursive)
					if err != nil {
						return t, err
					}
					t.Child = append(t.Child, recursiveTeams)
				}
			}
		}
	}
	return t, nil
}

func ListTeamByName(ctx context.Context, g *GitHubClient, repo repository.Repository, teamNames []string, child bool, recursive bool) ([]*github.Team, error) {
	var teams []*github.Team
	for _, teamName := range teamNames {
		team, err := TeamByName(ctx, g, repo, teamName, child, recursive)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team.Flatten()...)
	}
	return teams, nil
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
		team, err := g.GetTeamBySlug(ctx, repo.Owner, teamName)
		if err != nil {
			return nil, err
		}
		if team != nil && team.Parent != nil {
			parentRepos, err := g.ListTeamRepos(ctx, repo.Owner, *team.Parent.Slug)
			if err != nil {
				return nil, err
			}
			repos = CompareRepositories(repos, parentRepos).Left()
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
