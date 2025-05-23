package render

import (
	"github.com/google/go-github/v71/github"
	"github.com/olekukonko/tablewriter"
)

func (r *Renderer) RenderHovercard(hovercard *github.Hovercard) {
	if r.exporter != nil {
		r.RenderExportedData(hovercard)
		return
	}

	table := tablewriter.NewWriter(r.IO.Out)
	headers := []string{"MESSAGE", "OCTION"}
	table.SetHeader(headers)

	for _, context := range hovercard.Contexts {
		row := make([]string, len(headers))
		row[0] = ToString(context.Message)
		row[1] = ToString(context.Octicon)
		table.Append(row)
	}
	table.Render()
}
