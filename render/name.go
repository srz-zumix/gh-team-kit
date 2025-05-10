package render

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v71/github"
)

func getNames(items any) []string {
	switch v := items.(type) {
	case []*github.Repository:
		names := make([]string, len(v))
		for i, item := range v {
			names[i] = *item.FullName
		}
		return names
	case []*github.Team:
		names := make([]string, len(v))
		for i, item := range v {
			names[i] = *item.Slug
		}
		return names
	case []*github.User:
		names := make([]string, len(v))
		for i, item := range v {
			names[i] = *item.Login
		}
		return names
	default:
		return nil
	}
}

func (r *Renderer) RenderNames(items any) {
	names := getNames(items)
	if r.exporter != nil {
		r.RenderExportedData(names)
		return
	}

	if names == nil {
		return
	}

	defer fmt.Fprintln(r.IO.Out, strings.Join(names, "\n"))
}
