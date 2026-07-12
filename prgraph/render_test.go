package prgraph

import (
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

func TestMermaidNodeID(t *testing.T) {
	a := mermaidNodeID("ci-test")
	b := mermaidNodeID("ci_test")
	if a == b {
		t.Errorf("expected distinct IDs for ci-test and ci_test, got %q", a)
	}
}
