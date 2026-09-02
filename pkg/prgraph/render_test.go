package prgraph

import (
	"regexp"
	"strings"
	"testing"

	"github.com/srz-zumix/go-gh-extension/pkg/render"
)

func buildTestGraph() *Graph {
	g := NewGraph()
	alice := g.AddNode(NodeTypeUser, "alice")
	bob := g.AddNode(NodeTypeUser, "bob")
	team := g.AddNode(NodeTypeTeam, "backend")
	label := g.AddNode(NodeTypeLabel, "bug")
	file := g.AddNode(NodeTypeFile, "cmd/root.go")
	dir := g.AddNode(NodeTypeDirectory, "cmd")
	g.AddEdge(alice, bob, RelationReviewed)
	g.AddEdge(alice, bob, RelationReviewed)
	g.AddEdge(bob, team, RelationMemberOf)
	g.AddEdge(bob, label, RelationLabeled)
	g.AddEdge(bob, file, RelationChanged)
	g.AddEdge(file, dir, RelationInDirectory)
	return g
}

func TestRenderMermaid(t *testing.T) {
	sr := render.NewStringRenderer(nil)
	if err := Render(&sr.Renderer, "mermaid", buildTestGraph()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := sr.Stdout.String()
	for _, want := range []string{
		"graph LR",
		`(["alice"])`,
		`[["backend"]]`,
		`{{"bug"}}`,
		`["cmd/root.go"]`,
		`[("cmd")]`,
		`"reviewed (2)"`,
		`"member-of"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("mermaid output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	sr := render.NewStringRenderer(nil)
	if err := Render(&sr.Renderer, "markdown", buildTestGraph()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := sr.Stdout.String()
	if !strings.HasPrefix(out, "```mermaid\n") || !strings.Contains(out, "\n```\n") {
		t.Errorf("markdown output should wrap mermaid in a code fence:\n%s", out)
	}
}

func TestRenderDot(t *testing.T) {
	sr := render.NewStringRenderer(nil)
	if err := Render(&sr.Renderer, "dot", buildTestGraph()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := sr.Stdout.String()
	for _, want := range []string{
		"digraph {",
		`"user:alice" [label="alice" shape=ellipse]`,
		`"team:backend" [label="backend" shape=doubleoctagon]`,
		`"label:bug" [label="bug" shape=hexagon]`,
		`"file:cmd/root.go" [label="cmd/root.go" shape=box]`,
		`"directory:cmd" [label="cmd" shape=folder]`,
		`"user:alice" -> "user:bob" [label="reviewed (2)"]`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dot output missing %q:\n%s", want, out)
		}
	}
}

func TestRenderUnsupportedFormat(t *testing.T) {
	sr := render.NewStringRenderer(nil)
	if err := Render(&sr.Renderer, "png", buildTestGraph()); err == nil {
		t.Errorf("expected error for unsupported format")
	}
}

func TestMermaidCompactNodeIDs(t *testing.T) {
	sr := render.NewStringRenderer(nil)
	if err := Render(&sr.Renderer, "mermaid", buildTestGraph()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := sr.Stdout.String()

	// Node definition lines must use short sequential identifiers (n0, n1, ...)
	// so that long file/directory paths are not repeated across edges, which
	// would inflate the diagram beyond Mermaid's maximum text size.
	defRe := regexp.MustCompile(`(?m)^    (n\d+)`)
	defined := make(map[string]bool)
	for _, m := range defRe.FindAllStringSubmatch(out, -1) {
		defined[m[1]] = true
	}
	if len(defined) == 0 {
		t.Fatalf("expected compact node identifiers in output:\n%s", out)
	}

	// Every edge endpoint must reference a defined node identifier.
	edgeRe := regexp.MustCompile(`(?m)^    (n\d+) -- ".*" --> (n\d+)$`)
	edges := edgeRe.FindAllStringSubmatch(out, -1)
	if len(edges) == 0 {
		t.Fatalf("expected edges referencing compact node identifiers:\n%s", out)
	}
	for _, e := range edges {
		if !defined[e[1]] || !defined[e[2]] {
			t.Errorf("edge references undefined node id: %q -> %q", e[1], e[2])
		}
	}
}

func TestFormatWeight(t *testing.T) {
	tests := []struct {
		weight float64
		want   string
	}{
		{weight: 1, want: "1"},
		{weight: 2, want: "2"},
		{weight: 0.5, want: "0.5"},
		{weight: 2.5, want: "2.5"},
		{weight: 0.125, want: "0.13"},
		{weight: 1.0000001, want: "1"},
		{weight: 0, want: "0"},
	}
	for _, tt := range tests {
		if got := formatWeight(tt.weight); got != tt.want {
			t.Errorf("formatWeight(%v) = %q, want %q", tt.weight, got, tt.want)
		}
	}
}

func TestEdgeLabelShowsFractionalWeights(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		want   string
	}{
		{name: "a single occurrence hides the weight", weight: 1, want: "reviewed"},
		{name: "a whole weight keeps its integer form", weight: 3, want: "reviewed (3)"},
		{name: "a decayed weight is shown", weight: 0.5, want: "reviewed (0.5)"},
		{name: "a rounded weight is shown", weight: 1.004, want: "reviewed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edge := &Edge{From: "a", To: "b", Relation: RelationReviewed, Weight: tt.weight}
			if got := edgeLabel(edge); got != tt.want {
				t.Errorf("edgeLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}
