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

func (g *GitHubClient) GetUserHovercard(ctx context.Context, username string, subjectType, subjectId string) (*github.Hovercard, error) {
	var opts *github.HovercardOptions
	if subjectType != "" || subjectId != "" {
		opts = &github.HovercardOptions{
			SubjectType: subjectType,
			SubjectID:   subjectId,
		}
	}
	user, _, err := g.client.Users.GetHovercard(ctx, username, opts)
	if err != nil {
		return nil, err
	}
	return user, nil
}
