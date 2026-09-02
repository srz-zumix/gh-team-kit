package prgraph

import (
	"math"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
)

func TestFileWeightByBasis(t *testing.T) {
	tests := []struct {
		name      string
		weightBy  string
		additions int
		deletions int
		want      float64
	}{
		{name: "default counts the file once", weightBy: "", additions: 10, deletions: 5, want: 1},
		{name: "occurrences counts the file once", weightBy: WeightByOccurrences, additions: 10, deletions: 5, want: 1},
		{name: "lines sums both sides", weightBy: WeightByLines, additions: 10, deletions: 5, want: 15},
		{name: "additions ignores deletions", weightBy: WeightByAdditions, additions: 10, deletions: 5, want: 10},
		{name: "deletions ignores additions", weightBy: WeightByDeletions, additions: 10, deletions: 5, want: 5},
		{name: "deletion-only file weighted by additions is zero", weightBy: WeightByAdditions, additions: 0, deletions: 5, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &collector{opts: Options{WeightBy: tt.weightBy}}
			if got := c.fileWeight(tt.additions, tt.deletions); got != tt.want {
				t.Errorf("fileWeight(%d, %d) = %v, want %v", tt.additions, tt.deletions, got, tt.want)
			}
		})
	}
}

func TestDecayFactor(t *testing.T) {
	reference := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		halfLife float64
		created  time.Time
		want     float64
	}{
		{name: "no decay without a half-life", halfLife: 0, created: reference.AddDate(0, 0, -90), want: 1},
		{name: "no decay for an unknown date", halfLife: 30, created: time.Time{}, want: 1},
		{name: "no decay at the reference time", halfLife: 30, created: reference, want: 1},
		{name: "no decay for a future pull request", halfLife: 30, created: reference.AddDate(0, 0, 10), want: 1},
		{name: "halved after one half-life", halfLife: 30, created: reference.AddDate(0, 0, -30), want: 0.5},
		{name: "quartered after two half-lives", halfLife: 30, created: reference.AddDate(0, 0, -60), want: 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &collector{opts: Options{HalfLife: tt.halfLife}, referenceTime: reference}
			got := c.decayFactor(tt.created)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("decayFactor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecayReferenceUsesUntil(t *testing.T) {
	until := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	if got := decayReference(Options{Until: &until}); !got.Equal(until) {
		t.Errorf("decayReference() = %v, want %v", got, until)
	}
	if got := decayReference(Options{}); got.IsZero() {
		t.Error("expected decayReference to fall back to the current time")
	}
}

func TestIsCreatedStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: "added", want: true},
		{status: "copied", want: true},
		{status: "renamed", want: false},
		{status: "modified", want: false},
		{status: "removed", want: false},
		{status: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isCreatedStatus(tt.status); got != tt.want {
				t.Errorf("isCreatedStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsAllowedUser(t *testing.T) {
	tests := []struct {
		name       string
		allowUsers []string
		login      string
		want       bool
	}{
		{name: "every user is allowed without an allowlist", allowUsers: nil, login: "alice", want: true},
		{name: "a listed user is allowed", allowUsers: []string{"alice"}, login: "alice", want: true},
		{name: "the allowlist is case-insensitive", allowUsers: []string{"ALICE"}, login: "alice", want: true},
		{name: "an unlisted user is rejected", allowUsers: []string{"alice"}, login: "bob", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &collector{allowUsers: tt.allowUsers}
			if got := c.isAllowedUser(tt.login); got != tt.want {
				t.Errorf("isAllowedUser(%q) = %v, want %v", tt.login, got, tt.want)
			}
		})
	}
}

func TestIsHiddenUserAppliesTheAllowlist(t *testing.T) {
	c := &collector{allowUsers: []string{"alice"}}
	if c.isHiddenUser("alice") {
		t.Error("did not expect an allowed user to be hidden")
	}
	if !c.isHiddenUser("bob") {
		t.Error("expected a user outside the allowlist to be hidden")
	}
}

func TestIsDeletedFile(t *testing.T) {
	c := &collector{}
	if c.isDeletedFile("gone.go") {
		t.Error("did not expect a deleted file without a resolved tree")
	}
	c.trackedPaths = map[string]bool{"kept.go": true}
	if c.isDeletedFile("kept.go") {
		t.Error("did not expect a tracked file to be reported as deleted")
	}
	if !c.isDeletedFile("gone.go") {
		t.Error("expected an untracked file to be reported as deleted")
	}
}

func TestAddCoChangeEdges(t *testing.T) {
	newContributions := func(g *Graph, weights map[string]float64) (map[string]*pathContribution, []string) {
		contributions := make(map[string]*pathContribution)
		var order []string
		for _, name := range []string{"a.go", "b.go", "c.go"} {
			weight, ok := weights[name]
			if !ok {
				continue
			}
			node := g.AddNode(NodeTypeFile, name)
			contributions[node.ID] = &pathContribution{node: node, weight: weight}
			order = append(order, node.ID)
		}
		return contributions, order
	}

	t.Run("pairs every path once", func(t *testing.T) {
		c := &collector{graph: NewGraph(), opts: Options{CoChange: true}}
		contributions, order := newContributions(c.graph, map[string]float64{"a.go": 1, "b.go": 1, "c.go": 1})
		c.addCoChangeEdges(1, contributions, order)
		if len(c.graph.Edges) != 3 {
			t.Fatalf("expected 3 co-change edges, got %d", len(c.graph.Edges))
		}
		for _, edge := range c.graph.Edges {
			if edge.Relation != RelationCoChanged {
				t.Errorf("expected a co-changed relation, got %q", edge.Relation)
			}
			if edge.From > edge.To {
				t.Errorf("expected the smaller node ID as the source, got %q -> %q", edge.From, edge.To)
			}
		}
	})

	t.Run("uses the smaller contribution as the pair weight", func(t *testing.T) {
		c := &collector{graph: NewGraph(), opts: Options{CoChange: true}}
		contributions, order := newContributions(c.graph, map[string]float64{"a.go": 100, "b.go": 3})
		c.addCoChangeEdges(1, contributions, order)
		if len(c.graph.Edges) != 1 {
			t.Fatalf("expected 1 co-change edge, got %d", len(c.graph.Edges))
		}
		if c.graph.Edges[0].Weight != 3 {
			t.Errorf("expected weight 3, got %v", c.graph.Edges[0].Weight)
		}
	})

	t.Run("is disabled by default", func(t *testing.T) {
		c := &collector{graph: NewGraph()}
		contributions, order := newContributions(c.graph, map[string]float64{"a.go": 1, "b.go": 1})
		c.addCoChangeEdges(1, contributions, order)
		if len(c.graph.Edges) != 0 {
			t.Errorf("expected no co-change edges when disabled, got %d", len(c.graph.Edges))
		}
	})

	t.Run("skips pull requests above the path limit", func(t *testing.T) {
		c := &collector{graph: NewGraph(), opts: Options{CoChange: true, CoChangeMaxFiles: 2}}
		contributions, order := newContributions(c.graph, map[string]float64{"a.go": 1, "b.go": 1, "c.go": 1})
		c.addCoChangeEdges(1, contributions, order)
		if len(c.graph.Edges) != 0 {
			t.Errorf("expected no co-change edges above the limit, got %d", len(c.graph.Edges))
		}
	})

	t.Run("needs at least two paths", func(t *testing.T) {
		c := &collector{graph: NewGraph(), opts: Options{CoChange: true}}
		contributions, order := newContributions(c.graph, map[string]float64{"a.go": 1})
		c.addCoChangeEdges(1, contributions, order)
		if len(c.graph.Edges) != 0 {
			t.Errorf("expected no co-change edges for a single path, got %d", len(c.graph.Edges))
		}
	})
}

func TestAddUserEdgeUsesTheGivenWeight(t *testing.T) {
	c := &collector{graph: NewGraph(), involvedUsers: make(map[string]map[string]bool)}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	target := c.graph.AddNode(NodeTypeUser, "bob")

	c.addUserEdge(repo, "alice", target, RelationReviewed, 0.25)

	if len(c.graph.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(c.graph.Edges))
	}
	if c.graph.Edges[0].Weight != 0.25 {
		t.Errorf("expected weight 0.25, got %v", c.graph.Edges[0].Weight)
	}
}

func TestParseUserAllowlist(t *testing.T) {
	content := []byte("# owners\n\n@alice\nbob \n  # trailing comment\nAlice\n@\n")
	got := ParseUserAllowlist(content)
	want := []string{"alice", "bob"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected %v, got %v", want, got)
		}
	}
	if logins := ParseUserAllowlist([]byte("# only comments\n\n")); len(logins) != 0 {
		t.Errorf("expected no logins, got %v", logins)
	}
}
