package render

import (
	"strings"

	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
)

type userFiledGetter func(user *github.User) string
type userFiledGetters struct {
	Func map[string]userFiledGetter
}

func NewUserFieldGetters() *userFiledGetters {
	return &userFiledGetters{
		Func: map[string]userFiledGetter{
			"USERNAME": func(user *github.User) string {
				return *user.Login
			},
			"EMAIL": func(user *github.User) string {
				return ToString(user.Email)
			},
			"ROLE": func(user *github.User) string {
				return ToString(user.RoleName)
			},
			"SUSPENDED": func(user *github.User) string {
				return ToString(user.SuspendedAt != nil)
			},
			"URL": func(user *github.User) string {
				return ToString(user.HTMLURL)
			},
			"NAME": func(user *github.User) string {
				return ToString(user.Name)
			},
			"TEAM": func(user *github.User) string {
				if user.InheritedFrom == nil {
					return ""
				}
				names := make([]string, len(user.InheritedFrom))
				for i, team := range user.InheritedFrom {
					names[i] = ToString(team.Slug)
				}
				return strings.Join(names, ", ")
			},
		},
	}
}

func (u *userFiledGetters) GetField(user *github.User, field string) string {
	if getter, ok := u.Func[field]; ok {
		return getter(user)
	}
	return ""
}

func (r *Renderer) RenderUsers(users []*github.User, headers []string) {
	if r.exporter != nil {
		r.RenderExportedData(users)
		return
	}

	getter := NewUserFieldGetters()
	table := tablewriter.NewWriter(r.IO.Out)
	table.SetHeader(headers)

	for _, user := range users {
		row := make([]string, len(headers))
		for i, header := range headers {
			row[i] = getter.GetField(user, header)
		}
		table.Append(row)
	}
	table.Render()
}

func (r *Renderer) RenderUserWithRole(users []*github.User) {
	headers := []string{"USERNAME", "ROLE"}
	r.RenderUsers(users, headers)
}

func (r *Renderer) RenderUserDetails(users []*github.User) {
	headers := []string{"USERNAME", "ROLE", "EMAIL", "SUSPENDED"}
	r.RenderUsers(users, headers)
}
