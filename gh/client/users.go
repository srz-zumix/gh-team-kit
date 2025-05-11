package client

import (
	"context"

	"github.com/google/go-github/v71/github"
)

// GetUser retrieves a user by their username.
func (g *GitHubClient) GetUser(ctx context.Context, username string) (*github.User, error) {
	user, _, err := g.client.Users.Get(ctx, username)
	if err != nil {
		return nil, err
	}
	return user, nil
}
