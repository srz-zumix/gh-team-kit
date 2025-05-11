package gh

import (
	"context"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

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
					recursiveTeams.Team = childTeam
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

// ListTeams is a wrapper function that uses a Repository object to call either ListTeams or ListTeamsByRepo.
func ListTeams(ctx context.Context, g *GitHubClient, repo repository.Repository) ([]*github.Team, error) {
	if repo.Name != "" {
		return g.ListRepositoryTeams(ctx, repo.Owner, repo.Name)
	}
	return g.ListTeams(ctx, repo.Owner)
}

// GetTeamBySlug retrieves a team by its name.
func GetTeamBySlug(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) (*github.Team, error) {
	return g.GetTeamBySlug(ctx, repo.Owner, teamSlug)
}

// FindTeamBySlug retrieves a team by its name.
func FindTeamBySlug(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) (*github.Team, error) {
	return g.FindTeamBySlug(ctx, repo.Owner, teamSlug)
}

// IsExistsTeam checks if a team exists by its slug.
func IsExistsTeam(ctx context.Context, client *GitHubClient, repository repository.Repository, teamSlug string) (bool, error) {
	t, err := FindTeamBySlug(ctx, client, repository, teamSlug)
	if err != nil {
		return false, err
	}
	return t != nil, nil
}

// ListChildTeams is a wrapper function that calls the ListChildTeams API.
func ListChildTeams(ctx context.Context, g *GitHubClient, repo repository.Repository, parentSlug string) ([]*github.Team, error) {
	return g.ListChildTeams(ctx, repo.Owner, parentSlug)
}

// HasChildTeams checks if a team has child teams.
func HasChildTeams(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) (bool, error) {
	childTeams, err := ListChildTeams(ctx, g, repo, teamSlug)
	if err != nil {
		return false, err
	}
	return len(childTeams) > 0, nil
}

// RemoveTeamRepo is a wrapper function to remove a repository from a team.
func RemoveTeamRepo(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) error {
	return g.RemoveTeamRepo(ctx, repo.Owner, teamSlug, repo.Owner, repo.Name)
}

// AddTeamRepo is a wrapper function to add a repository to a team.
func AddTeamRepo(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, permission string) error {
	return g.AddTeamRepo(ctx, repo.Owner, teamSlug, repo.Owner, repo.Name, permission)
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

// ListRepositoryTeams is a wrapper for the GitHubClient's ListRepositoryTeams method.
func ListRepositoryTeams(ctx context.Context, g *GitHubClient, repo repository.Repository) ([]*github.Team, error) {
	return g.ListRepositoryTeams(ctx, repo.Owner, repo.Name)
}

// CreateTeam creates a new team in the specified organization.
func CreateTeam(ctx context.Context, g *GitHubClient, repo repository.Repository, name string, description string, privacy string, enableNotification bool, parentTeamSlug string) (*github.Team, error) {
	newTeam := &github.NewTeam{
		Name:         name,
		Description:  &description,
		Privacy:      &privacy,
		ParentTeamID: nil, // ParentTeamSlug will be handled differently
	}

	if parentTeamSlug != "" {
		parentTeam, err := g.GetTeamBySlug(ctx, repo.Owner, parentTeamSlug)
		if err != nil {
			return nil, err
		}
		if parentTeam != nil && parentTeam.ID != nil {
			newTeam.ParentTeamID = parentTeam.ID
		}
	}

	notificationSetting := "notifications_disabled"
	if enableNotification {
		notificationSetting = "notifications_enabled"
	}
	newTeam.NotificationSetting = &notificationSetting

	return g.CreateTeam(ctx, repo.Owner, newTeam)
}

// DeleteTeam deletes a team by its slug in the specified repository.
func DeleteTeam(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string) error {
	return g.DeleteTeamBySlug(ctx, repo.Owner, teamSlug)
}

// UpdateTeam updates the details of a team in the specified repository.
func UpdateTeam(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, name *string, description *string, privacy *string, enableNotification *bool, parentTeamSlug *string) (*github.Team, error) {
	team := &github.NewTeam{
		Name:         teamSlug,
		Description:  description,
		Privacy:      privacy,
		ParentTeamID: nil, // ParentTeamSlug will be handled differently
	}

	if name != nil {
		team.Name = *name
	}
	if enableNotification != nil {
		notificationSetting := "notifications_disabled"
		if *enableNotification {
			notificationSetting = "notifications_enabled"
		}
		team.NotificationSetting = &notificationSetting
	}

	removeParent := false
	if parentTeamSlug != nil {
		if *parentTeamSlug != "" {
			parentTeam, err := g.GetTeamBySlug(ctx, repo.Owner, *parentTeamSlug)
			if err != nil {
				return nil, err
			}
			if parentTeam != nil && parentTeam.ID != nil {
				team.ParentTeamID = parentTeam.ID
			}
		} else {
			// If parentTeamSlug is empty, remove the parent association
			removeParent = true
		}
	}

	return g.UpdateTeam(ctx, repo.Owner, teamSlug, team, removeParent)
}

// RenameTeam renames a team by its slug in the specified repository.
func RenameTeam(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, newName string) (*github.Team, error) {
	return UpdateTeam(ctx, g, repo, teamSlug, &newName, nil, nil, nil, nil)
}
