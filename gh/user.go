package gh

import (
	"context"
	"reflect"
	"slices"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

func GetLoginUser(ctx context.Context, g *GitHubClient) (*github.User, error) {
	user, err := g.GetUser(ctx, "")
	if err != nil {
		return nil, err
	}
	return user, nil
}

func UpdateUsers(ctx context.Context, g *GitHubClient, users []*github.User) ([]*github.User, error) {
	for _, user := range users {
		userDetails, err := g.GetUser(ctx, *user.Login)
		if err != nil {
			return nil, err
		}
		if userDetails != nil {
			userValue := reflect.ValueOf(user).Elem()
			userDetailsValue := reflect.ValueOf(userDetails).Elem()
			for i := 0; i < userValue.NumField(); i++ {
				field := userValue.Field(i)
				if field.IsZero() {
					field.Set(userDetailsValue.Field(i))
				}
			}
		}
	}
	return users, nil
}

func CollectSuspendedUsers(users []*github.User) []*github.User {
	var suspendedUsers []*github.User
	for _, user := range users {
		if user.SuspendedAt != nil {
			suspendedUsers = append(suspendedUsers, user)
		}
	}
	return suspendedUsers
}

func ExcludeSuspendedUsers(users []*github.User) []*github.User {
	var suspendedUsers []*github.User
	for _, user := range users {
		if user.SuspendedAt == nil {
			suspendedUsers = append(suspendedUsers, user)
		}
	}
	return suspendedUsers
}

func ExcludeOrganizationAdmins(ctx context.Context, g *GitHubClient, repo repository.Repository, users []*github.User) ([]*github.User, error) {
	admins, err := ListOrgMembers(ctx, g, repo, []string{"admin"}, false)
	if err != nil {
		return nil, err
	}
	var filteredUsers []*github.User
	for _, user := range users {
		if slices.ContainsFunc(admins, func(admin *github.User) bool {
			return *admin.ID == *user.ID
		}) {
			continue
		}
		filteredUsers = append(filteredUsers, user)
	}
	return filteredUsers, nil
}

// DetectUserTeams adds team information to each user (collaborator) for a repository
func DetectUserTeams(ctx context.Context, g *GitHubClient, repo repository.Repository, users []*github.User) ([]*github.User, error) {
	teams, err := g.ListRepositoryTeams(ctx, repo.Owner, repo.Name)
	if err != nil {
		return users, err
	}
	for _, team := range teams {
		members, err := g.ListTeamMembers(ctx, repo.Owner, team.GetSlug(), "")
		if err != nil {
			continue // skip error team
		}
		for _, user := range users {
			if GetPermissionName(user.Permissions) == GetPermissionName(team.Permissions) {
				if slices.ContainsFunc(members, func(m *github.User) bool {
					return *m.ID == *user.ID
				}) {
					if user.InheritedFrom == nil {
						user.InheritedFrom = make([]*github.Team, 0)
					}
					user.InheritedFrom = append(user.InheritedFrom, team)
				}
			}
		}
	}
	return users, nil
}
