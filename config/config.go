package config

type OrganizationConfig struct {
	Teams     []TeamConfig     `yaml:"teams" json:"teams"`
	Hierarchy []*TeamHierarchy `yaml:"hierarchy,omitempty" json:"hierarchy,omitempty"`
}

type TeamConfig struct {
	Name                string                  `yaml:"name" json:"name"`
	Slug                string                  `yaml:"slug" json:"slug"`
	Description         string                  `yaml:"description,omitempty" json:"description,omitempty"`
	Privacy             string                  `yaml:"privacy,omitempty" json:"privacy,omitempty"`
	ParentTeam          *string                 `yaml:"parent_team_slug,omitempty" json:"parent_team_slug,omitempty"`
	NotificationSetting string                  `yaml:"notification_setting,omitempty" json:"notification_setting,omitempty"`
	Maintainers         []string                `yaml:"maintainers,omitempty" json:"maintainers,omitempty"`
	Members             []string                `yaml:"members,omitempty" json:"members,omitempty"`
	CodeReviewSettings  *TeamCodeReviewSettings `yaml:"code_review_settings,omitempty" json:"code_review_settings,omitempty"`
}

type TeamCodeReviewSettings struct {
	Enabled                      bool     `yaml:"enabled" json:"enabled"`
	Algorithm                    string   `yaml:"algorithm" json:"algorithm"`
	TeamMemberCount              int      `yaml:"member_count" json:"member_count"`
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
