package render

import (
	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
)

func (r *Renderer) RenderCustomOrgRoles(roles []*github.CustomOrgRoles) {
	if r.exporter != nil {
		r.RenderExportedData(roles)
		return
	}

	if len(roles) == 0 {
		r.WriteLine("no roles found")
		return
	}

	headers := []string{"ID", "NAME", "DESCRIPTION"}
	table := tablewriter.NewWriter(r.IO.Out)
	table.SetHeader(headers)

	for _, role := range roles {
		data := []string{
			ToString(role.ID),
			*role.Name,
			*role.Description,
		}
		table.Append(data)
	}

	table.Render()
}
