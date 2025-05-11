package gh

import (
	"testing"

	"github.com/google/go-github/v71/github"
)

func TestFindRepository(t *testing.T) {
	repo1 := &github.Repository{ID: github.Ptr(int64(1))}
	repo2 := &github.Repository{ID: github.Ptr(int64(2))}
	repos := []*github.Repository{repo1, repo2}

	target := &github.Repository{ID: github.Ptr(int64(1))}
	result := FindRepository(target, repos)

	if result == nil || *result.ID != *target.ID {
		t.Errorf("Expected repository with ID %d, got nil or different ID", *target.ID)
	}

	nonExistent := &github.Repository{ID: github.Ptr(int64(3))}
	result = FindRepository(nonExistent, repos)

	if result != nil {
		t.Errorf("Expected nil for non-existent repository, got %v", result)
	}
}

func TestFilterRepositoriesByNames(t *testing.T) {
	repo1 := &github.Repository{FullName: github.Ptr("owner/repo1")}
	repo2 := &github.Repository{FullName: github.Ptr("owner/repo2")}
	repos := []*github.Repository{repo1, repo2}

	names := []string{"repo1"}
	owner := "owner"
	filtered := FilterRepositoriesByNames(repos, names, owner)

	if len(filtered) != 1 || *filtered[0].FullName != "owner/repo1" {
		t.Errorf("Expected filtered repository to be 'owner/repo1', got %v", filtered)
	}

	names = []string{"repo3"}
	filtered = FilterRepositoriesByNames(repos, names, owner)

	if len(filtered) != 0 {
		t.Errorf("Expected no repositories to be filtered, got %v", filtered)
	}
}

func TestFilterTeamByNames(t *testing.T) {
	team1 := &github.Team{Slug: github.Ptr("team1")}
	team2 := &github.Team{Slug: github.Ptr("team2")}
	teams := []*github.Team{team1, team2}

	slugs := []string{"team1"}
	filtered := FilterTeamByNames(teams, slugs)

	if len(filtered) != 1 || *filtered[0].Slug != "team1" {
		t.Errorf("Expected filtered team to be 'team1', got %v", filtered)
	}

	slugs = []string{"team3"}
	filtered = FilterTeamByNames(teams, slugs)

	if len(filtered) != 0 {
		t.Errorf("Expected no teams to be filtered, got %v", filtered)
	}
}
