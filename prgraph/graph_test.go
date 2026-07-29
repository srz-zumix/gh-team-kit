package prgraph

import (
	"testing"
)

func TestAddNodeDeduplicates(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "alice")
	if a != b {
		t.Errorf("expected the same node instance for duplicate AddNode calls")
	}
	if len(g.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(g.Nodes))
	}
}

func TestAddNodeDistinguishesTypes(t *testing.T) {
	g := NewGraph()
	g.AddNode(NodeTypeUser, "core")
	g.AddNode(NodeTypeTeam, "core")
	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes for same name with different types, got %d", len(g.Nodes))
	}
}

func TestAddEdgeAccumulatesWeight(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	g.AddEdge(a, b, RelationReviewed)
	edge := g.AddEdge(a, b, RelationReviewed)
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if edge.Weight != 2 {
		t.Errorf("expected weight 2, got %d", edge.Weight)
	}
}

func TestAddEdgeDistinguishesRelations(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	g.AddEdge(a, b, RelationReviewed)
	g.AddEdge(a, b, RelationCommented)
	if len(g.Edges) != 2 {
		t.Errorf("expected 2 edges for different relations, got %d", len(g.Edges))
	}
}

func TestNodeID(t *testing.T) {
	if got := NodeID(NodeTypeFile, "cmd/root.go"); got != "file:cmd/root.go" {
		t.Errorf("unexpected node ID: %s", got)
	}
}
