package render

import (
	"slices"

	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
)

type teamFiledGetter func(user *github.Team) string
type teamFiledGetters struct {
	Func map[string]teamFiledGetter
}

func NewTeamFieldGetters() *teamFiledGetters {
	return &teamFiledGetters{
		Func: map[string]teamFiledGetter{
			"NAME": func(user *github.Team) string {
				return *user.Name
			},
			"DESCRIPTION": func(user *github.Team) string {
				return ToString(user.Description)
			},
			"MEMBER_COUNT": func(user *github.Team) string {
				return ToString(user.MembersCount)
			},
			"REPOS_COUNT": func(user *github.Team) string {
				return ToString(user.ReposCount)
			},
			"PERMISSION": func(user *github.Team) string {
				return ToString(user.Permission)
			},
			"PRIVACY": func(user *github.Team) string {
				return ToString(user.Privacy)
			},
			"URL": func(user *github.Team) string {
				return ToString(user.HTMLURL)
			},
			"PARENT_SLUG": func(user *github.Team) string {
				if user.Parent == nil {
					return ""
				}
				return ToString(user.Parent.Slug)
			},
		},
	}
}

func (u *teamFiledGetters) GetField(user *github.Team, field string) string {
	if getter, ok := u.Func[field]; ok {
		return getter(user)
	}
	return ""
}

func (r *Renderer) RenderTeams(teams []*github.Team, headers []string) {
	if r.exporter != nil {
		r.RenderExportedData(teams)
		return
	}

	if len(teams) == 0 {
		r.WriteLine("no teams found")
		return
	}

	headers = slices.DeleteFunc(headers, func(s string) bool {
		switch s {
		case "MEMBER_COUNT":
			return teams[0].MembersCount == nil
		case "REPOS_COUNT":
			return teams[0].ReposCount == nil
		}
		return false
	})

	getter := NewTeamFieldGetters()
	table := tablewriter.NewWriter(r.IO.Out)
	table.SetHeader(headers)

	for _, team := range teams {
		row := make([]string, len(headers))
		for i, header := range headers {
			row[i] = getter.GetField(team, header)
		}
		table.Append(row)
	}

	table.Render()
}

func (r *Renderer) RenderTeamsDefault(teams []*github.Team) {
	headers := []string{"NAME", "DESCRIPTION", "MEMBER_COUNT", "REPOS_COUNT", "PRIVACY", "PARENT_SLUG"}
	r.RenderTeams(teams, headers)
}

func (r *Renderer) RenderTeamsWithPermission(teams []*github.Team) {
	headers := []string{"NAME", "DESCRIPTION", "MEMBER_COUNT", "REPOS_COUNT", "PRIVACY", "PERMISSION", "PARENT_SLUG"}
	r.RenderTeams(teams, headers)
}
