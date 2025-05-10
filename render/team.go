package render

import (
	"fmt"

	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
)

func (r *Renderer) RenderTeam(teams []*github.Team) {
	if r.exporter != nil {
		r.RenderExportedData(teams)
		return
	}

	if len(teams) == 0 {
		defer fmt.Fprintln(r.IO.Out, "No teams")
		return
	}

	headers := []string{"NAME", "DESCRIPTION"}
	hasCount := false
	if teams[0].MembersCount != nil && teams[0].ReposCount != nil {
		headers = append(headers, "MEMBER_COUNT", "REPOS_COUNT")
		hasCount = true
	}
	headers = append(headers, "PARENT_SLUG")

	table := tablewriter.NewWriter(r.IO.Out)
	table.SetHeader(headers)

	for _, team := range teams {
		data := []string{
			*team.Name,
			*team.Description,
		}
		if hasCount {
			data = append(data,
				fmt.Sprintf("%d", *team.MembersCount),
				fmt.Sprintf("%d", *team.ReposCount),
			)
		}
		parentSlug := ""
		if team.Parent != nil && team.Parent.Slug != nil {
			parentSlug = *team.Parent.Slug
		}
		data = append(data, parentSlug)
		table.Append(data)
	}

	table.Render()
}
