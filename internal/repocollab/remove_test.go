package repocollab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"

	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

func TestBuildRemoveChanges(t *testing.T) {
	users := []*github.User{
		{Login: new("alice"), Permissions: &github.RepositoryPermissions{Pull: new(true)}},
		{Login: new("bob")},
	}
	changes := BuildRemoveChanges("github.com/o/r", users)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	for i, want := range []string{"alice", "bob"} {
		c := changes[i]
		if c.Target != want || c.Action != gh.SyncChangeRemove || c.Destination != "github.com/o/r" {
			t.Errorf("unexpected change[%d]: %+v", i, c)
		}
	}
}

func TestBuildRemoveChangesEmpty(t *testing.T) {
	if got := BuildRemoveChanges("github.com/o/r", nil); len(got) != 0 {
		t.Errorf("expected no changes, got %d", len(got))
	}
}

const suspendedEMULogin = "0123456789abcdef0123456789abcdef_SHORT"

func newFixtureClient(t *testing.T, handler http.HandlerFunc) *gh.GitHubClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)
	client, err := gh.NewGitHubClient()
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

func TestPlanRemoveSuspendedCollaborators(t *testing.T) {
	var affiliation string
	client := newFixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/repos/o/r/collaborators") {
			http.Error(w, "unexpected request", http.StatusNotFound)
			return
		}
		affiliation = r.URL.Query().Get("affiliation")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"login":"alice","permissions":{"push":true}},
			{"login":"` + suspendedEMULogin + `","permissions":{"pull":true}}
		]`))
	})
	repo := repository.Repository{Host: "github.com", Owner: "o", Name: "r"}
	changes, err := PlanRemoveSuspendedCollaborators(context.Background(), client, repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if affiliation != "direct" {
		t.Errorf("expected affiliation=direct, got %q", affiliation)
	}
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %+v", len(changes), changes)
	}
	if c := changes[0]; c.Target != suspendedEMULogin || c.Action != gh.SyncChangeRemove {
		t.Errorf("unexpected change: %+v", c)
	}
}

func TestPlanRemoveSuspendedCollaboratorsListError(t *testing.T) {
	client := newFixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	})
	repo := repository.Repository{Host: "github.com", Owner: "o", Name: "r"}
	changes, err := PlanRemoveSuspendedCollaborators(context.Background(), client, repo)
	if err == nil {
		t.Fatal("expected error")
	}
	if changes != nil {
		t.Errorf("expected no changes on error, got %+v", changes)
	}
}
