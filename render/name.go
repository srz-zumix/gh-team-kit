package render

import (
	"fmt"

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
		r.exporter.Write(r.IO, names)
		return
	}

	if names == nil {
		return
	}
	for _, name := range names {
		fmt.Fprintln(r.IO.Out, name)
	}
}
