package prgraph

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// Render writes the graph using the configured exporter, if any.
// Without an exporter, format selects the output: "mermaid", "markdown", or "dot".
func Render(r *render.Renderer, format string, graph *Graph) error {
	if r.HasExporter() {
		return r.RenderExportedData(graph)
	}
	switch strings.ToLower(format) {
	case "dot":
		return renderDot(r, graph)
	case "markdown":
		r.WriteLine("```mermaid")
		err := renderMermaid(r, graph)
		r.WriteLine("```")
		return err
	case "mermaid":
		return renderMermaid(r, graph)
	default:
		return fmt.Errorf("unsupported graph format: %s", format)
	}
}

// renderMermaid writes the graph as a Mermaid flowchart with typed node
// shapes and edge labels including the relation name and weight. Nodes are
// referenced by short sequential identifiers (n0, n1, ...) instead of encoding
// each node's full name into its identifier: long file and directory paths
// would otherwise be repeated across every edge, bloating the diagram text
// until it exceeds Mermaid's maximum text size and fails to render.
func renderMermaid(r *render.Renderer, graph *Graph) error {
	r.WriteLine("graph LR")
	ids := make(map[string]string, len(graph.Nodes))
	for i, node := range graph.Nodes {
		id := fmt.Sprintf("n%d", i)
		ids[node.ID] = id
		r.WriteLine(fmt.Sprintf("    %s%s", id, mermaidNodeShape(node)))
	}
	for _, edge := range graph.Edges {
		r.WriteLine(fmt.Sprintf("    %s -- \"%s\" --> %s",
			ids[edge.From],
			edgeLabel(edge),
			ids[edge.To],
		))
	}
	return nil
}

// renderDot writes the graph as a Graphviz DOT digraph with typed node
// shapes and edge labels including the relation name and weight.
func renderDot(r *render.Renderer, graph *Graph) error {
	r.WriteLine("digraph {")
	for _, node := range graph.Nodes {
		r.WriteLine(fmt.Sprintf("    %s [label=%s shape=%s]",
			dotQuote(node.ID),
			dotQuote(node.Name),
			dotNodeShape(node.Type),
		))
	}
	for _, edge := range graph.Edges {
		r.WriteLine(fmt.Sprintf("    %s -> %s [label=%s]",
			dotQuote(edge.From),
			dotQuote(edge.To),
			dotQuote(edgeLabel(edge)),
		))
	}
	r.WriteLine("}")
	return nil
}

// edgeLabel builds the display label for an edge, appending the weight unless
// it is exactly one. Fractional weights below one are shown, since decayed or
// line-based contributions are meaningful even when smaller than a single
// occurrence.
func edgeLabel(edge *Edge) string {
	weight := formatWeight(edge.Weight)
	if weight == "1" {
		return edge.Relation
	}
	return fmt.Sprintf("%s (%s)", edge.Relation, weight)
}

// formatWeight renders a weight rounded to two decimal places without trailing
// zeros, so that whole weights keep their original integer representation.
func formatWeight(weight float64) string {
	return strconv.FormatFloat(math.Round(weight*100)/100, 'f', -1, 64)
}

// mermaidNodeShape returns the Mermaid node definition for the given node,
// using a distinct shape per node type.
func mermaidNodeShape(node *Node) string {
	name := strings.ReplaceAll(node.Name, "\"", "#quot;")
	switch node.Type {
	case NodeTypeUser:
		return fmt.Sprintf("([\"%s\"])", name)
	case NodeTypeTeam:
		return fmt.Sprintf("[[\"%s\"]]", name)
	case NodeTypeLabel:
		return fmt.Sprintf("{{\"%s\"}}", name)
	case NodeTypeDirectory:
		return fmt.Sprintf("[(\"%s\")]", name)
	case NodeTypeSubmodule:
		return fmt.Sprintf("[/\"%s\"/]", name)
	default: // NodeTypeFile
		return fmt.Sprintf("[\"%s\"]", name)
	}
}

// dotNodeShape returns the Graphviz shape name for a node type.
func dotNodeShape(t NodeType) string {
	switch t {
	case NodeTypeUser:
		return "ellipse"
	case NodeTypeTeam:
		return "doubleoctagon"
	case NodeTypeLabel:
		return "hexagon"
	case NodeTypeDirectory:
		return "folder"
	case NodeTypeSubmodule:
		return "component"
	default: // NodeTypeFile
		return "box"
	}
}

// dotQuote returns a DOT-safe quoted string by escaping backslashes and double quotes.
func dotQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}
