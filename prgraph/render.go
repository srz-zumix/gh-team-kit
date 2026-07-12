package prgraph

import (
	"fmt"
	"strings"

	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

// Render writes the graph to the renderer in the specified format.
// Supported formats: "mermaid", "markdown", and "dot".
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
// shapes and edge labels including the relation name and weight.
func renderMermaid(r *render.Renderer, graph *Graph) error {
	r.WriteLine("graph LR")
	for _, node := range graph.Nodes {
		r.WriteLine(fmt.Sprintf("    %s%s", mermaidNodeID(node.ID), mermaidNodeShape(node)))
	}
	for _, edge := range graph.Edges {
		r.WriteLine(fmt.Sprintf("    %s -- \"%s\" --> %s",
			mermaidNodeID(edge.From),
			edgeLabel(edge),
			mermaidNodeID(edge.To),
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

// edgeLabel builds the display label for an edge, appending the weight when greater than one.
func edgeLabel(edge *Edge) string {
	if edge.Weight > 1 {
		return fmt.Sprintf("%s (%d)", edge.Relation, edge.Weight)
	}
	return edge.Relation
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
	default: // NodeTypeFile
		return "box"
	}
}

// mermaidNodeID creates a collision-free Mermaid node identifier from a string.
// All non-alphanumeric characters are hex-encoded using their Unicode code point
// as six lowercase hexadecimal digits prefixed with '_', so that encoded
// sequences cannot be confused with adjacent alphanumeric characters.
func mermaidNodeID(name string) string {
	var b strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else {
			fmt.Fprintf(&b, "_%06x", c)
		}
	}
	return b.String()
}

// dotQuote returns a DOT-safe quoted string by escaping backslashes and double quotes.
func dotQuote(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}
