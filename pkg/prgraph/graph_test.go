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
		t.Errorf("expected weight 2, got %v", edge.Weight)
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

func TestFilterEdgesNoop(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	g.AddEdge(a, b, RelationReviewed)
	g.FilterEdges(nil, nil)
	if len(g.Edges) != 1 {
		t.Errorf("expected FilterEdges with no filters to keep all edges, got %d", len(g.Edges))
	}
}

func TestFilterEdgesInclude(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	c := g.AddNode(NodeTypeFile, "main.go")
	g.AddEdge(a, b, RelationReviewed)
	g.AddEdge(a, c, RelationChanged)
	g.FilterEdges([]string{RelationChanged}, nil)
	if len(g.Edges) != 1 || g.Edges[0].Relation != RelationChanged {
		t.Errorf("expected only %q edges to remain, got %+v", RelationChanged, g.Edges)
	}
}

func TestFilterEdgesExclude(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	c := g.AddNode(NodeTypeFile, "main.go")
	g.AddEdge(a, b, RelationReviewed)
	g.AddEdge(a, c, RelationChanged)
	g.FilterEdges(nil, []string{RelationReviewed})
	if len(g.Edges) != 1 || g.Edges[0].Relation != RelationChanged {
		t.Errorf("expected %q edges to be excluded, got %+v", RelationReviewed, g.Edges)
	}
}

func TestFilterMinWeightNoop(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	g.AddEdge(a, b, RelationReviewed)
	g.FilterMinWeight(0)
	if len(g.Edges) != 1 {
		t.Errorf("expected FilterMinWeight(0) to keep all edges, got %d", len(g.Edges))
	}
}

func TestFilterMinWeightRemovesLightEdges(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	c := g.AddNode(NodeTypeFile, "main.go")
	g.AddEdge(a, b, RelationReviewed)
	g.AddEdge(a, c, RelationChanged)
	g.AddEdge(a, c, RelationChanged)
	g.FilterMinWeight(2)
	if len(g.Edges) != 1 || g.Edges[0].Relation != RelationChanged {
		t.Errorf("expected only the weight-2 edge to remain, got %+v", g.Edges)
	}
}

func TestRemoveOrphanNodes(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeUser, "bob")
	dir := g.AddNode(NodeTypeDirectory, "src")
	g.AddEdge(a, b, RelationReviewed)
	g.AddEdge(a, dir, RelationChanged)
	g.FilterEdges(nil, []string{RelationChanged})
	g.RemoveOrphanNodes()
	if len(g.Nodes) != 2 {
		t.Errorf("expected the orphaned directory node to be removed, got %+v", g.Nodes)
	}
	if g.Node(dir.ID) != nil {
		t.Error("expected the orphaned node to be dropped from the index")
	}
}

func TestAddEdgeWeightAccumulates(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeFile, "cmd/root.go")
	g.AddEdgeWeight(a, b, RelationChanged, 0.5)
	edge := g.AddEdgeWeight(a, b, RelationChanged, 0.25)
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if edge.Weight != 0.75 {
		t.Errorf("expected weight 0.75, got %v", edge.Weight)
	}
}

func TestRoundWeights(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeFile, "cmd/root.go")
	edge := g.AddEdgeWeight(a, b, RelationChanged, 0.1)
	g.AddEdgeWeight(a, b, RelationChanged, 0.2)
	g.RoundWeights()
	if edge.Weight != 0.3 {
		t.Errorf("expected weight 0.3, got %v", edge.Weight)
	}
}

func TestFilterMinWeightAcceptsFractions(t *testing.T) {
	g := NewGraph()
	a := g.AddNode(NodeTypeUser, "alice")
	b := g.AddNode(NodeTypeFile, "a.go")
	c := g.AddNode(NodeTypeFile, "b.go")
	g.AddEdgeWeight(a, b, RelationChanged, 0.25)
	g.AddEdgeWeight(a, c, RelationChanged, 0.75)
	g.FilterMinWeight(0.5)
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].To != c.ID {
		t.Errorf("expected the heavier edge to survive, got %q", g.Edges[0].To)
	}
}
