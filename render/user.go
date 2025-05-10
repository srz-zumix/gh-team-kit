package render

import (
	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
)

func (r *Renderer) RenderUser(users []*github.User) {
	if r.exporter != nil {
		r.RenderExportedData(users)
		return
	}

	headers := []string{"USERNAME", "ROLE"}
	table := tablewriter.NewWriter(r.IO.Out)
	table.SetHeader(headers)

	for _, user := range users {
		row := []string{
			*user.Login,
			ToString(user.RoleName),
		}
		table.Append(row)
	}
	table.Render()
}

func (r *Renderer) RenderUserDetails(users []*github.User) {
	if r.exporter != nil {
		r.RenderExportedData(users)
		return
	}
	headers := []string{"USERNAME", "ROLE", "EMAIL", "SUSPENDED"}
	table := tablewriter.NewWriter(r.IO.Out)
	table.SetHeader(headers)

	for _, user := range users {
		row := []string{
			*user.Login,
			ToString(user.RoleName),
			ToString(user.Email),
			ToString(user.SuspendedAt != nil),
		}
		table.Append(row)
	}
	table.Render()
}
