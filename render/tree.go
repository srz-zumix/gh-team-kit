package render

import (
	"fmt"

	"github.com/ddddddO/gtree"
	"github.com/srz-zumix/gh-team-kit/gh"
)

func teamRootTree(rootName string, team gh.Team) *gtree.Node {
	if team.Team == nil {
		return gtree.NewRoot(rootName)
	}
	return nil
}

func teamTree(node *gtree.Node, team gh.Team) *gtree.Node {
	root := node
	if team.Team != nil {
		if node == nil {
			node = gtree.NewRoot(team.Team.GetSlug())
			root = node
		} else {
			node = node.Add(team.Team.GetSlug())
		}
	} else {
		if node == nil {
			return nil
		}
	}
	for _, child := range team.Child {
		teamTree(node, child)
	}
	return root
}

func (r *Renderer) RenderTeamTree(rootName string, team gh.Team) {
	if r.exporter != nil {
		r.RenderExportedData(team)
	}

	root := teamTree(teamRootTree(rootName, team), team)
	err := gtree.OutputFromRoot(r.IO.Out, root)
	if err != nil {
		fmt.Fprintf(r.IO.ErrOut, "Error rendering team tree: %v\n", err)
	}
}
