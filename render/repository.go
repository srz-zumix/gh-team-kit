package render

import (
	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
	"github.com/srz-zumix/gh-team-kit/gh"
)

func (r *Renderer) RenderRepository(repos []*github.Repository) {
	if r.exporter != nil {
		r.RenderExportedData(repos)
		return
	}
	headers := []string{"NAME", "PERMISSION", "VISIBILITY"}
	table := tablewriter.NewWriter(r.IO.Out)
	table.SetHeader(headers)

	for _, repo := range repos {
		permission := gh.GetRepositoryPermissions(repo)
		row := []string{
			*repo.FullName,
			permission,
			*repo.Visibility,
		}
		table.Append(row)
	}
	table.Render()
}
