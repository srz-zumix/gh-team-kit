package gh

import (
	"context"
	"reflect"

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
