package gh

import (
	"context"
	"reflect"
	"slices"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

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
