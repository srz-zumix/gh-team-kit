package gh

import (
	"context"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v71/github"
)

// GetTeamMembership retrieves the membership details of a user in a specific team.
func GetTeamMembership(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, username string) (*github.Membership, error) {
	return g.GetTeamMembership(ctx, repo.Owner, teamSlug, username)
}

// FindTeamMembership retrieves the membership details of a user in a specific team.
func FindTeamMembership(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, username string) (*github.Membership, error) {
	return g.FindTeamMembership(ctx, repo.Owner, teamSlug, username)
}

// ListTeamMembers retrieves all members of a specific team in the organization and filters them by roles if provided.
func ListTeamMembers(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, roles []string, membership bool) ([]*github.User, error) {
	members, err := g.ListTeamMembers(ctx, repo.Owner, teamSlug, GetTeamMembershipFilter(roles))
	if err != nil {
		return nil, err
	}

	if membership {
		for _, member := range members {
			membership, err := g.GetTeamMembership(ctx, repo.Owner, teamSlug, *member.Login)
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

// AddTeamMember is a wrapper function to add or update a team member.
func AddTeamMember(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, username string, role string, allowNonOrganizationMember bool) (*github.Membership, error) {
	if !allowNonOrganizationMember {
		membership, err := g.FindOrgMembership(ctx, repo.Owner, username)
		if err != nil {
			return nil, err
		}
		if membership == nil {
			return nil, fmt.Errorf("user '%s' is not a member of the organization '%s'", username, repo.Owner)
		}
	}
	return g.AddOrUpdateTeamMembership(ctx, repo.Owner, teamSlug, username, role)
}

func UpdateTeamMemberRole(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, username string, role string) (*github.Membership, error) {
	membership, err := g.FindTeamMembership(ctx, repo.Owner, teamSlug, username)
	if err != nil {
		return nil, fmt.Errorf("failed to find membership for '%s' in team '%s': %w", username, teamSlug, err)
	}
	if membership == nil {
		return nil, fmt.Errorf("user '%s' is not a member of team '%s'", username, teamSlug)
	}
	return g.AddOrUpdateTeamMembership(ctx, repo.Owner, teamSlug, username, role)
}

// RemoveTeamMember is a wrapper function to remove a user from a team.
func RemoveTeamMember(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, username string) error {
	return g.RemoveTeamMember(ctx, repo.Owner, teamSlug, username)
}
