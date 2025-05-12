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

// ListTeamsAssignedToRole retrieves teams assigned to a specific organization role.
func ListTeamsAssignedToRole(ctx context.Context, g *GitHubClient, repo repository.Repository, role string) ([]*github.Team, error) {
	roleID, err := GetRoleIDByName(ctx, g, repo, role)
	if err != nil {
		return nil, fmt.Errorf("failed to get role ID for role '%s' in organization '%s': %w", role, repo.Owner, err)
	}
	teams, err := g.ListTeamsAssignedToOrgRole(ctx, repo.Owner, *roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams for role '%s' in organization '%s': %w", role, repo.Owner, err)
	}
	return teams, nil
}

// ListRoles retrieves all roles available in the specified organization.
func ListOrgRoles(ctx context.Context, g *GitHubClient, repo repository.Repository) ([]*github.CustomOrgRoles, error) {
	roles, err := g.ListOrgRoles(ctx, repo.Owner)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles in organization '%s': %w", repo.Owner, err)
	}
	return roles.CustomRepoRoles, nil
}

// GetRoleIDByName retrieves the RoleID for a given Role name in the specified organization.
func GetRoleIDByName(ctx context.Context, g *GitHubClient, repo repository.Repository, roleName string) (*int64, error) {
	roles, err := ListOrgRoles(ctx, g, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles in organization '%s': %w", repo.Owner, err)
	}

	for _, role := range roles {
		if *role.Name == roleName {
			return role.ID, nil
		}
	}

	return nil, fmt.Errorf("role '%s' not found in organization '%s'", roleName, repo.Owner)
}

// ListUsersAssignedToOrgRole retrieves users assigned to a specific organization role.
func ListUsersAssignedToOrgRole(ctx context.Context, g *GitHubClient, repo repository.Repository, roleName string) ([]*github.User, error) {
	roles, err := ListOrgRoles(ctx, g, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles in organization '%s': %w", repo.Owner, err)
	}

	var users []*github.User
	for _, role := range roles {
		if roleName == "" || *role.Name == roleName {
			usersFromRole, err := g.ListUsersAssignedToOrgRole(ctx, repo.Owner, *role.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to list users for role '%s' in organization '%s': %w", *role.Name, repo.Owner, err)
			}
			for _, user := range users {
				user.RoleName = role.Name
			}
			users = append(users, usersFromRole...)
		}
	}

	return users, nil
}

// AssignOrgRoleToTeam assigns a specific organization role to a team.
func AssignOrgRoleToTeam(ctx context.Context, g *GitHubClient, repo repository.Repository, teamSlug string, role string) error {
	roleID, err := GetRoleIDByName(ctx, g, repo, role)
	if err != nil {
		return fmt.Errorf("failed to get role ID for role '%s' in organization '%s': %w", role, repo.Owner, err)
	}
	return g.AssignOrgRoleToTeam(ctx, repo.Owner, teamSlug, *roleID)
}
