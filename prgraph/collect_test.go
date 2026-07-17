package prgraph

import (
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/hmarr/codeowners"
)

func TestAddDirectoryChain(t *testing.T) {
	c := &collector{graph: NewGraph()}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "a/b/c.go")
	c.addDirectoryChain(repo, fileNode, "a/b/c.go")

	if c.graph.Node(NodeID(NodeTypeDirectory, "a/b")) == nil {
		t.Errorf("expected directory node a/b")
	}
	if c.graph.Node(NodeID(NodeTypeDirectory, "a")) == nil {
		t.Errorf("expected directory node a")
	}
	if len(c.graph.Edges) != 2 {
		t.Errorf("expected 2 containment edges, got %d", len(c.graph.Edges))
	}
}

func TestAddDirectoryChainRootFile(t *testing.T) {
	c := &collector{graph: NewGraph()}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "main.go")
	c.addDirectoryChain(repo, fileNode, "main.go")
	if len(c.graph.Edges) != 0 {
		t.Errorf("expected no directory edges for a root file, got %d", len(c.graph.Edges))
	}
}

func TestPathNamePrefixesForMultipleRepos(t *testing.T) {
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	single := &collector{graph: NewGraph(), multiRepo: false}
	if got := single.pathName(repo, "a/b.go"); got != "a/b.go" {
		t.Errorf("unexpected single-repo path name: %s", got)
	}
	multi := &collector{graph: NewGraph(), multiRepo: true}
	if got := multi.pathName(repo, "a/b.go"); got != "octo/repo:a/b.go" {
		t.Errorf("unexpected multi-repo path name: %s", got)
	}
}

func TestAddCodeownersEdges(t *testing.T) {
	ruleset, err := codeowners.ParseFile(strings.NewReader("*.go @octo/backend @alice\ndocs/ @bob\n"))
	if err != nil {
		t.Fatalf("failed to parse CODEOWNERS: %v", err)
	}
	c := &collector{graph: NewGraph()}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "cmd/root.go")
	c.addCodeownersEdges(repo, fileNode, "cmd/root.go", ruleset)

	if c.graph.Node(NodeID(NodeTypeTeam, "backend")) == nil {
		t.Errorf("expected team node for @octo/backend shortened to slug")
	}
	if c.graph.Node(NodeID(NodeTypeUser, "alice")) == nil {
		t.Errorf("expected user node for @alice")
	}
	if len(c.graph.Edges) != 2 {
		t.Errorf("expected 2 owned-by edges, got %d", len(c.graph.Edges))
	}
}

func TestAddCodeownersEdgesNilRuleset(t *testing.T) {
	c := &collector{graph: NewGraph()}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "cmd/root.go")
	c.addCodeownersEdges(repo, fileNode, "cmd/root.go", nil)
	if len(c.graph.Edges) != 0 {
		t.Errorf("expected no edges without a ruleset, got %d", len(c.graph.Edges))
	}
}

func TestTeamNameNamespacing(t *testing.T) {
	single := &collector{graph: NewGraph(), multiOwner: false}
	if got := single.teamName("octo", "backend"); got != "backend" {
		t.Errorf("single-owner team name = %q, want %q", got, "backend")
	}
	multi := &collector{graph: NewGraph(), multiOwner: true}
	if got := multi.teamName("octo", "backend"); got != "octo/backend" {
		t.Errorf("multi-owner team name = %q, want %q", got, "octo/backend")
	}
}

func TestDistinctOwners(t *testing.T) {
	repos := []repository.Repository{
		{Host: "github.com", Owner: "octo", Name: "a"},
		{Host: "github.com", Owner: "octo", Name: "b"},
	}
	if got := distinctOwners(repos); got != 1 {
		t.Errorf("distinctOwners (same owner) = %d, want 1", got)
	}
	repos = append(repos, repository.Repository{Host: "github.com", Owner: "acme", Name: "c"})
	if got := distinctOwners(repos); got != 2 {
		t.Errorf("distinctOwners (two owners) = %d, want 2", got)
	}
}

func TestAddCodeownersEdgesMultiOwner(t *testing.T) {
	ruleset, err := codeowners.ParseFile(strings.NewReader("*.go @octo/backend @other/backend\n"))
	if err != nil {
		t.Fatalf("failed to parse CODEOWNERS: %v", err)
	}
	c := &collector{graph: NewGraph(), multiOwner: true}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "cmd/root.go")
	c.addCodeownersEdges(repo, fileNode, "cmd/root.go", ruleset)

	if c.graph.Node(NodeID(NodeTypeTeam, "octo/backend")) == nil {
		t.Errorf("expected namespaced team node octo/backend in multi-owner mode")
	}
	if c.graph.Node(NodeID(NodeTypeTeam, "other/backend")) == nil {
		t.Errorf("expected namespaced team node other/backend in multi-owner mode")
	}
	if c.graph.Node(NodeID(NodeTypeTeam, "backend")) != nil {
		t.Errorf("did not expect a bare-slug team node in multi-owner mode")
	}
}

func TestTrimTeamOrg(t *testing.T) {
	tests := []struct {
		value, owner, want string
	}{
		{"octo/backend", "octo", "backend"},
		{"Octo/backend", "octo", "backend"},
		{"other/backend", "octo", "other/backend"},
		{"backend", "octo", "backend"},
	}
	for _, tt := range tests {
		if got := trimTeamOrg(tt.value, tt.owner); got != tt.want {
			t.Errorf("trimTeamOrg(%q, %q) = %q, want %q", tt.value, tt.owner, got, tt.want)
		}
	}
}
