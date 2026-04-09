package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type OrganizationConfig struct {
	Teams     []TeamConfig     `yaml:"teams" json:"teams"`
	Hierarchy []*TeamHierarchy `yaml:"hierarchy,omitempty" json:"hierarchy,omitempty"`
}

type TeamRepositoryPermission struct {
	Name       string `yaml:"name" json:"name"`
	Permission string `yaml:"permission,omitempty" json:"permission,omitempty"`
}

type TeamConfig struct {
	Name                string                     `yaml:"name" json:"name"`
	Slug                string                     `yaml:"slug" json:"slug"`
	Description         string                     `yaml:"description,omitempty" json:"description,omitempty"`
	Privacy             string                     `yaml:"privacy,omitempty" json:"privacy,omitempty"`
	ParentTeam          *string                    `yaml:"parent_team_slug,omitempty" json:"parent_team_slug,omitempty"`
	NotificationSetting string                     `yaml:"notification_setting,omitempty" json:"notification_setting,omitempty"`
	Maintainers         []string                   `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	Members             []string                   `yaml:"members,omitempty" json:"members,omitempty"`
	Group               string                     `yaml:"group,omitempty" json:"group,omitempty"`
	OrgRoles            []string                   `yaml:"org_roles,omitempty" json:"org_roles,omitempty"`
	CodeReviewSettings  *TeamCodeReviewSettings    `yaml:"code_review_settings,omitempty" json:"code_review_settings,omitempty"`
	Repositories        []TeamRepositoryPermission `yaml:"repositories,omitempty" json:"repositories,omitempty"`
}

type TeamCodeReviewSettings struct {
	Enabled                      bool     `yaml:"enabled" json:"enabled"`
	Algorithm                    string   `yaml:"algorithm" json:"algorithm"`
	TeamMemberCount              int      `yaml:"team_member_count" json:"team_member_count"`
	NotifyTeam                   bool     `yaml:"notify_team" json:"notify_team"`
	ExcludedTeamMembers          []string `yaml:"excluded_team_members" json:"excluded_team_members"`
	IncludeChildTeamMembers      *bool    `yaml:"include_child_team_members" json:"include_child_team_members"`
	CountMembersAlreadyRequested *bool    `yaml:"count_members_already_requested" json:"count_members_already_requested"`
	RemoveTeamRequest            *bool    `yaml:"remove_team_request" json:"remove_team_request"`
}

type TeamHierarchy struct {
	Slug  string           `yaml:"slug" json:"slug"`
	Child []*TeamHierarchy `yaml:"child,omitempty" json:"child,omitempty"`
}

func (c *OrganizationConfig) FindTeamConfigBySlug(slug string) *TeamConfig {
	for i, team := range c.Teams {
		if team.Slug == slug {
			return &c.Teams[i]
		}
	}
	return nil
}

func (cfg *OrganizationConfig) WriteFile(output string) (err error) {
	f, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer func() {
		closeErr := f.Close()
		if err == nil {
			err = closeErr
		} else if closeErr != nil {
			err = fmt.Errorf("write error: %w; error closing file: %v", err, closeErr)
		}
	}()
	return cfg.Write(f)
}

func (cfg *OrganizationConfig) Write(w io.Writer) (err error) {
	encoder := yaml.NewEncoder(w)
	defer func() {
		closeErr := encoder.Close()
		if err == nil {
			err = closeErr
		} else if closeErr != nil {
			err = fmt.Errorf("write error: %w; error closing encoder: %v", err, closeErr)
		}
	}()
	return encoder.Encode(cfg)
}
