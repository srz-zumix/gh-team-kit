package prgraph

import (
	"context"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/hmarr/codeowners"
	ignore "github.com/sabhiram/go-gitignore"
)

func TestCollectPullRequestExcludesAuthor(t *testing.T) {
	c := &collector{
		graph:          NewGraph(),
		excludeAuthors: []string{"ALICE"},
	}
	activity := pullRequestActivity{author: "alice"}

	if err := c.collectPullRequest(context.Background(), repository.Repository{}, nil, activity, nil); err != nil {
		t.Fatalf("collectPullRequest returned an error: %v", err)
	}
	if len(c.graph.Nodes) != 0 {
		t.Errorf("expected no nodes for an excluded author, got %d", len(c.graph.Nodes))
	}
}

func TestIsExcludedAuthorAndIsHiddenUserAreIndependent(t *testing.T) {
	// A login placed only in hideUsers must not cause its own pull requests to
	// be skipped, and a login placed only in excludeAuthors must not hide its
	// non-author participation elsewhere.
	c := &collector{
		graph:          NewGraph(),
		excludeAuthors: []string{"relay-bot"},
		hideUsers:      []string{"comment-bot"},
	}
	if !c.isExcludedAuthor("relay-bot") {
		t.Error("expected relay-bot to be an excluded author")
	}
	if c.isHiddenUser("relay-bot") {
		t.Error("did not expect relay-bot to be hidden")
	}
	if c.isExcludedAuthor("comment-bot") {
		t.Error("did not expect comment-bot to be an excluded author")
	}
	if !c.isHiddenUser("comment-bot") {
		t.Error("expected comment-bot to be hidden")
	}
}

func TestAddUserEdgeHidesUser(t *testing.T) {
	c := &collector{
		graph:     NewGraph(),
		hideUsers: []string{"alice"},
	}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	authorNode := c.graph.AddNode(NodeTypeUser, "bob")

	c.addUserEdge(repo, "Alice", authorNode, RelationReviewed, 1)

	if c.graph.Node(NodeID(NodeTypeUser, "Alice")) != nil {
		t.Error("did not expect a node for a hidden user")
	}
	if len(c.graph.Edges) != 0 {
		t.Errorf("expected no edges for a hidden user, got %d", len(c.graph.Edges))
	}
}

func TestCombineLogins(t *testing.T) {
	if got := combineLogins(nil, nil); len(got) != 0 {
		t.Errorf("combineLogins(nil, nil) = %v, want empty", got)
	}
	got := combineLogins([]string{"alice"}, []string{"bob", "carol"})
	want := []string{"alice", "bob", "carol"}
	if len(got) != len(want) {
		t.Fatalf("combineLogins = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("combineLogins[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsExcludedFile(t *testing.T) {
	c := &collector{
		graph:        NewGraph(),
		excludeFiles: ignore.CompileIgnoreLines("*.md", "vendor/**"),
	}
	tests := []struct {
		filename string
		want     bool
	}{
		{"README.md", true},
		{"vendor/pkg/file.go", true},
		{"cmd/root.go", false},
	}
	for _, tt := range tests {
		if got := c.isExcludedFile(tt.filename); got != tt.want {
			t.Errorf("isExcludedFile(%q) = %v, want %v", tt.filename, got, tt.want)
		}
	}
}

func TestIsExcludedFileNilMatcher(t *testing.T) {
	c := &collector{graph: NewGraph()}
	if c.isExcludedFile("README.md") {
		t.Error("expected no exclusion without configured patterns")
	}
}

func TestAddDirectoryChain(t *testing.T) {
	c := &collector{graph: NewGraph()}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "a/b/c.go")
	c.addDirectoryChain(repo, fileNode, "a/b/c.go", 1)

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
	c.addDirectoryChain(repo, fileNode, "main.go", 1)
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
	c.addCodeownersEdges(repo, fileNode, "cmd/root.go", ruleset, 1)

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

func TestAddCodeownersEdgesExcludesUser(t *testing.T) {
	ruleset, err := codeowners.ParseFile(strings.NewReader("*.go @alice @bob\n"))
	if err != nil {
		t.Fatalf("failed to parse CODEOWNERS: %v", err)
	}
	c := &collector{
		graph:     NewGraph(),
		hideUsers: []string{"ALICE"},
	}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "cmd/root.go")

	c.addCodeownersEdges(repo, fileNode, "cmd/root.go", ruleset, 1)

	if c.graph.Node(NodeID(NodeTypeUser, "alice")) != nil {
		t.Error("did not expect a CODEOWNERS node for an excluded user")
	}
	if c.graph.Node(NodeID(NodeTypeUser, "bob")) == nil {
		t.Error("expected a CODEOWNERS node for a non-excluded user")
	}
	if len(c.graph.Edges) != 1 {
		t.Errorf("expected 1 owned-by edge, got %d", len(c.graph.Edges))
	}
}

func TestAddCodeownersEdgesNilRuleset(t *testing.T) {
	c := &collector{graph: NewGraph()}
	repo := repository.Repository{Owner: "octo", Name: "repo"}
	fileNode := c.graph.AddNode(NodeTypeFile, "cmd/root.go")
	c.addCodeownersEdges(repo, fileNode, "cmd/root.go", nil, 1)
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
	c.addCodeownersEdges(repo, fileNode, "cmd/root.go", ruleset, 1)

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

func TestMatchesLabelFilters(t *testing.T) {
	tests := []struct {
		name             string
		labels           []string
		include, exclude []string
		want             bool
	}{
		{"no filters", []string{"bug"}, nil, nil, true},
		{"include matches", []string{"bug", "auto-merge"}, []string{"bug"}, nil, true},
		{"include does not match", []string{"bug"}, []string{"feature"}, nil, false},
		{"exclude matches", []string{"auto-merge"}, nil, []string{"auto-merge"}, false},
		{"exclude does not match", []string{"bug"}, nil, []string{"auto-merge"}, true},
		{"exclude wins over include", []string{"bug", "auto-merge"}, []string{"bug"}, []string{"auto-merge"}, false},
		{"case-insensitive", []string{"Bug"}, []string{"bug"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesLabelFilters(tt.labels, tt.include, tt.exclude); got != tt.want {
				t.Errorf("matchesLabelFilters(%v, %v, %v) = %v, want %v", tt.labels, tt.include, tt.exclude, got, tt.want)
			}
		})
	}
}

func TestMatchesBranchPattern(t *testing.T) {
	tests := []struct {
		pattern, branch string
		want            bool
	}{
		{"", "main", true},
		{"main", "main", true},
		{"main", "develop", false},
		{"release/*", "release/1.0", true},
		{"release/*", "release/1.0/hotfix", false},
		{"[", "main", false}, // invalid pattern never matches
	}
	for _, tt := range tests {
		if got := matchesBranchPattern(tt.pattern, tt.branch); got != tt.want {
			t.Errorf("matchesBranchPattern(%q, %q) = %v, want %v", tt.pattern, tt.branch, got, tt.want)
		}
	}
}

func TestMatchesAnyBranchPattern(t *testing.T) {
	patterns := []string{"relay/*", "cherry-pick/*"}
	if !matchesAnyBranchPattern("relay/main", patterns) {
		t.Error("expected relay/main to match relay/*")
	}
	if matchesAnyBranchPattern("feature/foo", patterns) {
		t.Error("did not expect feature/foo to match any pattern")
	}
}

func TestIsBotLogin(t *testing.T) {
	tests := []struct {
		login string
		want  bool
	}{
		{"dependabot[bot]", true},
		{"DEPENDABOT[BOT]", true},
		{"alice", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isBotLogin(tt.login); got != tt.want {
			t.Errorf("isBotLogin(%q) = %v, want %v", tt.login, got, tt.want)
		}
	}
}

func TestNoBotsExcludesAndHidesBotLogins(t *testing.T) {
	c := &collector{
		graph: NewGraph(),
		opts:  Options{NoBots: true},
	}
	if !c.isExcludedAuthor("github-actions[bot]") {
		t.Error("expected a [bot]-suffixed login to be an excluded author when --no-bots is set")
	}
	if !c.isHiddenUser("github-actions[bot]") {
		t.Error("expected a [bot]-suffixed login to be hidden when --no-bots is set")
	}
	if c.isExcludedAuthor("alice") {
		t.Error("did not expect a human login to be excluded by --no-bots")
	}
}

func TestIsIncludedFile(t *testing.T) {
	c := &collector{}
	if !c.isIncludedFile("anything.go") {
		t.Error("expected isIncludedFile to allow everything when no --include-file patterns are configured")
	}
	c.includeFiles = compileIncludeFiles([]string{"src/**"})
	if !c.isIncludedFile("src/main.go") {
		t.Error("expected src/main.go to be included")
	}
	if c.isIncludedFile("docs/readme.md") {
		t.Error("expected docs/readme.md to not be included")
	}
}

func TestTruncatePathDepth(t *testing.T) {
	tests := []struct {
		filename string
		depth    int
		want     string
	}{
		{"a/b/c/file.go", 2, "a/b"},
		{"a/b/c/file.go", 10, "a/b/c/file.go"},
		{"file.go", 1, "file.go"},
	}
	for _, tt := range tests {
		if got := truncatePathDepth(tt.filename, tt.depth); got != tt.want {
			t.Errorf("truncatePathDepth(%q, %d) = %q, want %q", tt.filename, tt.depth, got, tt.want)
		}
	}
}

func TestGroupByPattern(t *testing.T) {
	tests := []struct {
		filename, pattern, want string
		wantOK                  bool
	}{
		{"LocalPackages/BattleEngine/Scripts/Foo.cs", "LocalPackages/*", "LocalPackages/BattleEngine", true},
		{"Assets/Foo.cs", "LocalPackages/*", "", false},
		{"LocalPackages/Foo.cs", "LocalPackages/*/Scripts", "", false},
	}
	for _, tt := range tests {
		got, ok := groupByPattern(tt.filename, tt.pattern)
		if ok != tt.wantOK || got != tt.want {
			t.Errorf("groupByPattern(%q, %q) = (%q, %v), want (%q, %v)", tt.filename, tt.pattern, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestGroupedPath(t *testing.T) {
	c := &collector{opts: Options{Depth: 1, GroupBy: []string{"LocalPackages/*", "Assets/*/*"}}}
	tests := []struct {
		filename   string
		want       string
		wantFolded bool
	}{
		{"LocalPackages/BattleEngine/Foo.cs", "LocalPackages/BattleEngine", true},
		{"Assets/Scripts/Battle/Foo.cs", "Assets/Scripts/Battle", true},
		{"docs/guide/readme.md", "docs", true},
		{"LocalPackages/Foo.cs", "LocalPackages/Foo.cs", false},
		{"README.md", "README.md", false},
	}
	for _, tt := range tests {
		got, folded := c.groupedPath(tt.filename)
		if got != tt.want || folded != tt.wantFolded {
			t.Errorf("groupedPath(%q) = (%q, %v), want (%q, %v)", tt.filename, got, folded, tt.want, tt.wantFolded)
		}
	}
}
