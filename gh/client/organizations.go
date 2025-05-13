package client

import (
	"context"

	"github.com/google/go-github/v71/github"
)

// ListOrgMembers retrieves all members of the specified organization.
func (g *GitHubClient) ListOrgMembers(ctx context.Context, org string, role string) ([]*github.User, error) {
	var allMembers []*github.User
	opt := &github.ListMembersOptions{
		Role:        role,
		ListOptions: github.ListOptions{PerPage: 50},
	}

	for {
		members, resp, err := g.client.Organizations.ListMembers(ctx, org, opt)
		if err != nil {
			return nil, err
		}
		allMembers = append(allMembers, members...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allMembers, nil
}

// GetOrgMembership retrieves the membership details of a user in the organization.
func (g *GitHubClient) GetOrgMembership(ctx context.Context, owner string, username string) (*github.Membership, error) {
	membership, _, err := g.client.Organizations.GetOrgMembership(ctx, username, owner)
	if err != nil {
		return nil, err
	}

	return membership, nil
}

// FindOrgMembership retrieves the membership details of a user in the organization.
// If the user is not a member, it returns nil without an error.
func (g *GitHubClient) FindOrgMembership(ctx context.Context, owner string, username string) (*github.Membership, error) {
	membership, resp, err := g.client.Organizations.GetOrgMembership(ctx, username, owner)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil, nil // User is not a member
		}
		return nil, err // Other errors
	}

	return membership, nil
}

// AddOrUpdateOrgMembership sets the membership details of a user in the organization.
func (g *GitHubClient) AddOrUpdateOrgMembership(ctx context.Context, org string, username string, role string) (*github.Membership, error) {
	membership, _, err := g.client.Organizations.EditOrgMembership(ctx, username, org, &github.Membership{Role: &role})
	if err != nil {
		return nil, err
	}
	return membership, nil
}

// RemoveOrgMember removes a member from the specified organization.
func (g *GitHubClient) RemoveOrgMember(ctx context.Context, org string, username string) error {
	_, err := g.client.Organizations.RemoveMember(ctx, org, username)
	if err != nil {
		return err
	}
	return nil
}

// CreateTeam creates a new team in the specified organization.
func (g *GitHubClient) CreateTeam(ctx context.Context, org string, team *github.NewTeam) (*github.Team, error) {
	createdTeam, _, err := g.client.Teams.CreateTeam(ctx, org, *team)
	if err != nil {
		return nil, err
	}
	return createdTeam, nil
}

// DeleteTeamBySlug deletes a team by its slug in the specified organization.
func (g *GitHubClient) DeleteTeamBySlug(ctx context.Context, org string, teamSlug string) error {
	_, err := g.client.Teams.DeleteTeamBySlug(ctx, org, teamSlug)
	if err != nil {
		return err
	}
	return nil
}

// UpdateTeam updates the details of a team in the specified repository.
func (g *GitHubClient) UpdateTeam(ctx context.Context, owner string, teamSlug string, team *github.NewTeam, removeParent bool) (*github.Team, error) {
	editedTeam, _, err := g.client.Teams.EditTeamBySlug(ctx, owner, teamSlug, *team, removeParent)
	if err != nil {
		return nil, err
	}
	return editedTeam, nil
}

// ListOrgTeams retrieves all teams assigned to an organization.
func (g *GitHubClient) ListOrgTeams(ctx context.Context, org string) ([]*github.Team, error) {
	var allTeams []*github.Team
	opt := &github.ListOptions{PerPage: 50}

	for {
		teams, resp, err := g.client.Teams.ListTeams(ctx, org, opt)
		if err != nil {
			return nil, err
		}
		allTeams = append(allTeams, teams...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allTeams, nil
}

// ListTeamsAssignedToOrgRole retrieves teams assigned to a specific organization role by roleID.
func (g *GitHubClient) ListTeamsAssignedToOrgRole(ctx context.Context, org string, roleID int64) ([]*github.Team, error) {
	var allTeams []*github.Team
	opt := &github.ListOptions{PerPage: 50}

	for {
		teams, resp, err := g.client.Organizations.ListTeamsAssignedToOrgRole(ctx, org, roleID, opt)
		if err != nil {
			return nil, err
		}
		allTeams = append(allTeams, teams...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allTeams, nil
}

// ListUsersAssignedToOrgRole retrieves users assigned to a specific organization role.
func (g *GitHubClient) ListUsersAssignedToOrgRole(ctx context.Context, org string, roleID int64) ([]*github.User, error) {
	var allUsers []*github.User
	opt := &github.ListOptions{
		PerPage: 50,
	}

	for {
		users, resp, err := g.client.Organizations.ListUsersAssignedToOrgRole(ctx, org, roleID, opt)
		if err != nil {
			return nil, err
		}
		allUsers = append(allUsers, users...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allUsers, nil
}

// ListOrgRoles retrieves all custom roles available in the specified organization.
func (g *GitHubClient) ListOrgRoles(ctx context.Context, org string) (*github.OrganizationCustomRoles, error) {
	roles, _, err := g.client.Organizations.ListRoles(ctx, org)
	if err != nil {
		return nil, err
	}

	return roles, nil
}

// AssignOrgRoleToTeam assigns a specific organization role to a team using the GitHub API.
func (g *GitHubClient) AssignOrgRoleToTeam(ctx context.Context, org string, teamSlug string, roleID int64) error {
	_, err := g.client.Organizations.AssignOrgRoleToTeam(ctx, org, teamSlug, roleID)
	return err
}

func (g *GitHubClient) RemoveOrgRoleFromTeam(ctx context.Context, org string, teamSlug string, roleID int64) error {
	_, err := g.client.Organizations.RemoveOrgRoleFromTeam(ctx, org, teamSlug, roleID)
	return err
}
