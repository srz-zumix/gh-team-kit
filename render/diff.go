package render

import (
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/fatih/color"
	"github.com/srz-zumix/gh-team-kit/gh"
)

func colorizeDiff(diff string) string {
	var result string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+ ") {
			result += color.GreenString(line) + "\n"
		} else if strings.HasPrefix(line, "- ") {
			result += color.RedString(line) + "\n"
		} else {
			result += line + "\n"
		}
	}
	return result
}

func getRepositoryPermissionDiffLines(diff gh.RepositoryPermissionsDiffs, left, right string) string {
	return diff.GetDiffLines(left, right)
}
func getTeamPermissionDiffLines(diff gh.TeamPermissionsDiffs, left, right repository.Repository) string {
	return diff.GetDiffLines(left, right)
}

func getDiffLines(diff any, left, right any) string {
	if d, ok := diff.(gh.RepositoryPermissionsDiffs); ok {
		return getRepositoryPermissionDiffLines(d, left.(string), right.(string))
	}
	if d, ok := diff.(gh.TeamPermissionsDiffs); ok {
		return getTeamPermissionDiffLines(d, left.(repository.Repository), right.(repository.Repository))
	}
	return ""
}

func (r *Renderer) RenderDiff(diff any, left, right any) {
	if r.exporter != nil {
		r.RenderExportedData(diff)
		return
	}

	diffLines := getDiffLines(diff, left, right)
	if r.Color {
		diffLines = colorizeDiff(diffLines)
	}
	r.WriteLine(diffLines)
}
