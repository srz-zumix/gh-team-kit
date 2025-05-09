package gh

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

// GetOrgMembership retrieves the membership details of a user in the specified organization.
func GetOrgMembership(ctx context.Context, g *GitHubClient, repo repository.Repository, username string) (*github.Membership, error) {
	return g.GetOrgMembership(ctx, repo.Owner, username)
}

// FindOrgMembership retrieves the membership details of a user in the specified organization.
func FindOrgMembership(ctx context.Context, g *GitHubClient, repo repository.Repository, username string) (*github.Membership, error) {
	return g.FindOrgMembership(ctx, repo.Owner, username)
}

// ListOrgMembers wraps the GitHubClient's ListOrgMembers function.
func ListOrgMembers(ctx context.Context, g *GitHubClient, repo repository.Repository, roles []string, membership bool) ([]*github.User, error) {
	roleFilter := GetOrgMembershipFilter(roles)
	members, err := g.ListOrgMembers(ctx, repo.Owner, roleFilter)
	if err != nil {
		return nil, err
	}

	if membership {
		for _, member := range members {
			membership, err := g.GetOrgMembership(ctx, repo.Owner, *member.Login)
			if err != nil {
				return nil, err
			}
			if membership != nil {
				member.RoleName = membership.Role
			}
		}
	}
	return members, nil
}

// RemoveOrgMember removes a member from the specified organization.
func RemoveOrgMember(ctx context.Context, g *GitHubClient, repo repository.Repository, username string) error {
	return g.RemoveOrgMember(ctx, repo.Owner, username)
}

// AddOrgMember adds a member to the specified organization with the given role.
func AddOrgMember(ctx context.Context, g *GitHubClient, repo repository.Repository, username string, role string) (*github.User, error) {
	membership, err := g.AddOrUpdateOrgMembership(ctx, repo.Owner, username, role)
	if err != nil {
		return nil, fmt.Errorf("failed to add '%s' to organization '%s': %w", username, repo.Owner, err)
	}
	membership.User.RoleName = membership.Role
	return membership.User, nil
}

func UpdateOrgMemberRole(ctx context.Context, g *GitHubClient, repo repository.Repository, username string, role string) (*github.User, error) {
	membership, err := g.FindOrgMembership(ctx, repo.Owner, username)
	if err != nil {
		return nil, fmt.Errorf("failed to find membership for '%s' in organization '%s': %w", username, repo.Owner, err)
	}
	if membership == nil {
		return nil, fmt.Errorf("user '%s' is not a member of organization '%s'", username, repo.Owner)
	}
	membership, err = g.AddOrUpdateOrgMembership(ctx, repo.Owner, username, role)
	if err != nil {
		return nil, fmt.Errorf("failed to update '%s' role in organization '%s': %w", username, repo.Owner, err)
	}
	membership.User.RoleName = membership.Role
	return membership.User, nil
}
