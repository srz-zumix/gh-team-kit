// Package prgraph analyzes pull request activity and builds a relationship
// graph between users, teams, labels, and code areas (files, directories,
// and CODEOWNERS owners).
package prgraph

import "fmt"

// NodeType identifies the kind of entity a graph node represents.
type NodeType string

const (
	NodeTypeUser      NodeType = "user"
	NodeTypeTeam      NodeType = "team"
	NodeTypeLabel     NodeType = "label"
	NodeTypeFile      NodeType = "file"
	NodeTypeDirectory NodeType = "directory"
)

// Relation names for graph edges.
const (
	RelationApproved         = "approved"
	RelationChangesRequested = "changes-requested"
	RelationReviewed         = "reviewed"
	RelationCommented        = "commented"
	RelationReviewCommented  = "review-commented"
	RelationReviewRequested  = "review-requested"
	RelationMemberOf         = "member-of"
	RelationChanged          = "changed"
	RelationInDirectory      = "in"
	RelationOwnedBy          = "owned-by"
	RelationLabeled          = "labeled"
)

// Node represents an entity (user, team, label, file, or directory) in the graph.
type Node struct {
	ID   string   `json:"id"`
	Type NodeType `json:"type"`
	Name string   `json:"name"`
}

// Edge represents a weighted, typed relationship between two nodes.
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
	Weight   int    `json:"weight"`
}

// Graph holds the nodes and edges of a PR activity relationship graph.
type Graph struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`

	nodeIndex map[string]*Node
	edgeIndex map[string]*Edge
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{
		nodeIndex: make(map[string]*Node),
		edgeIndex: make(map[string]*Edge),
	}
}

// NodeID builds the unique identifier for a node of the given type and name.
func NodeID(t NodeType, name string) string {
	return fmt.Sprintf("%s:%s", t, name)
}

// AddNode adds a node to the graph, returning the existing node if it is already present.
func (g *Graph) AddNode(t NodeType, name string) *Node {
	id := NodeID(t, name)
	if node, ok := g.nodeIndex[id]; ok {
		return node
	}
	node := &Node{ID: id, Type: t, Name: name}
	g.nodeIndex[id] = node
	g.Nodes = append(g.Nodes, node)
	return node
}

// AddEdge adds a directed edge between two nodes with the given relation,
// incrementing the weight if the same edge already exists.
func (g *Graph) AddEdge(from, to *Node, relation string) *Edge {
	key := fmt.Sprintf("%s|%s|%s", from.ID, to.ID, relation)
	if edge, ok := g.edgeIndex[key]; ok {
		edge.Weight++
		return edge
	}
	edge := &Edge{From: from.ID, To: to.ID, Relation: relation, Weight: 1}
	g.edgeIndex[key] = edge
	g.Edges = append(g.Edges, edge)
	return edge
}

// Node returns the node with the given ID, or nil if it does not exist.
func (g *Graph) Node(id string) *Node {
	return g.nodeIndex[id]
}
