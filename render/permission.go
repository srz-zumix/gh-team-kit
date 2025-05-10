package render

import (
	"fmt"

	"github.com/google/go-github/v71/github"
	"github.com/srz-zumix/gh-team-kit/gh"
)

func getPermissions(v any) map[string]bool {
	switch v := v.(type) {
	case *github.Repository:
		return v.Permissions
	case *github.Team:
		return v.Permissions
	case *github.User:
		return v.Permissions
	default:
		return nil
	}
}

func (r *Renderer) RenderPermission(v any) {
	var permissions = getPermissions(v)

	if r.exporter != nil {
		r.RenderExportedData(permissions)
		return
	}

	fmt.Fprintf(r.IO.Out, "%s\n", gh.GetPermissionName(permissions))
}
