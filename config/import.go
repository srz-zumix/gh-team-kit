package config

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/gh/client"
	"gopkg.in/yaml.v3"
)

type Importer struct {
	ctx    context.Context
	client *client.GitHubClient
	Owner  repository.Repository
}

func NewImporter(repository repository.Repository) (*Importer, error) {
	repository.Name = "" // Clear repository name to focus on organization level
	ctx := context.Background()
	client, err := gh.NewGitHubClientWithRepo(repository)
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub client: %w", err)
	}
	return &Importer{
		ctx:    ctx,
		client: client,
		Owner:  repository,
	}, nil
}

func (i *Importer) importTeam(organizationConfig *OrganizationConfig, teamHierarchies []*TeamHierarchy) ([]error, error) {
	errorList := []error{}
	for _, hierarchy := range teamHierarchies {
		teamConfig := organizationConfig.FindTeamConfigBySlug(hierarchy.Slug)
		if teamConfig == nil {
			return errorList, fmt.Errorf("team config not found for slug: %s", hierarchy.Slug)
		}

		_, err := gh.CreateOrUpdateTeam(i.ctx, i.client, i.Owner, teamConfig.Slug, &teamConfig.Name, teamConfig.Description, teamConfig.Privacy, teamConfig.NotificationSetting, teamConfig.ParentTeam)
		if err != nil {
			return errorList, fmt.Errorf("error creating or updating team %s: %w", teamConfig.Slug, err)
		}

		_, err = gh.AddTeamMembers(i.ctx, i.client, i.Owner, teamConfig.Slug, teamConfig.Members, gh.TeamMembershipRoleMember, true)
		if err != nil {
			errorList = append(errorList, err)
		}

		_, err = gh.AddTeamMembers(i.ctx, i.client, i.Owner, teamConfig.Slug, teamConfig.Maintainers, gh.TeamMembershipRoleMaintainer, true)
		if err != nil {
			errorList = append(errorList, err)
		}

		allMembers := append(teamConfig.Members, teamConfig.Maintainers...)
		err = gh.RemoveTeamMembersOther(i.ctx, i.client, i.Owner, teamConfig.Slug, allMembers)
		if err != nil {
			errorList = append(errorList, err)
		}

		childErrorList, err := i.importTeam(organizationConfig, hierarchy.Child)
		if err != nil {
			return errorList, err
		}
		errorList = append(errorList, childErrorList...)
	}
	return errorList, nil
}

func (i *Importer) Import(organizationConfig *OrganizationConfig) error {
	errorList, err := i.importTeam(organizationConfig, organizationConfig.Hierarchy)
	if err != nil {
		return err
	}
	if len(errorList) > 0 {
		return fmt.Errorf("encountered errors during import: %v", errorList)
	}
	return nil
}

func (i *Importer) ReadFile(input string) (c *OrganizationConfig, err error) {
	f, err := os.Open(input)
	if err != nil {
		return nil, fmt.Errorf("error opening input file: %w", err)
	}
	defer func() {
		err = f.Close()
	}()
	return i.Read(f)
}

func (i *Importer) Read(r io.Reader) (*OrganizationConfig, error) {
	var cfg OrganizationConfig
	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
