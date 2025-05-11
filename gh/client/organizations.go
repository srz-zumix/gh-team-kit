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
