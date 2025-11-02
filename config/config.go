package config

type OrganizationConfig struct {
	Teams     []TeamConfig     `yaml:"teams"`
	Hierarchy []*TeamHierarchy `yaml:"hierarchy,omitempty"`
}

type TeamConfig struct {
	Name                string   `yaml:"name"`
	Slug                string   `yaml:"slug"`
	Description         string   `yaml:"description,omitempty"`
	Privacy             string   `yaml:"privacy,omitempty"`
	ParentTeam          *string  `yaml:"parent_team_slug,omitempty"`
	NotificationSetting string   `yaml:"notification_setting,omitempty"`
	Maintainers         []string `yaml:"maintainers,omitempty"`
	Members             []string `yaml:"members,omitempty"`
}

type TeamHierarchy struct {
	Slug  string           `yaml:"slug"`
	Child []*TeamHierarchy `yaml:"child,omitempty"`
}

func (c *OrganizationConfig) FindTeamConfigBySlug(slug string) *TeamConfig {
	for _, team := range c.Teams {
		if team.Slug == slug {
			return &team
		}
	}
	return nil
}
