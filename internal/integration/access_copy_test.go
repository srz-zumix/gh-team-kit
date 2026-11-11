package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	teamcmd "github.com/srz-zumix/gh-team-kit/cmd"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

func TestAccessCopyProcess(t *testing.T) {
	if os.Getenv("GH_TEAM_KIT_TEST_ACCESS_COPY") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		fmt.Fprintln(os.Stderr, "missing command arguments")
		os.Exit(2)
	}
	cmd := &cobra.Command{Use: "gh-team-kit"}
	cmdflags.AddPersistentFlags(cmd)
	cmd.AddCommand(teamcmd.NewAccessCmd())
	cmd.SetArgs(append([]string{"access", "copy"}, os.Args[separator+1:]...))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func runAccessCopy(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	testArgs := append([]string{"-test.run=^TestAccessCopyProcess$", "--"}, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], testArgs...)
	cmd.Env = append(os.Environ(), "GH_TEAM_KIT_TEST_ACCESS_COPY=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("access copy process timed out: %v", ctx.Err())
	}
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	t.Fatalf("failed to run access copy process: %v", err)
	return "", "", -1
}

type copyRequest struct {
	method     string
	path       string
	permission string
}

type copyFixture struct {
	mu       sync.Mutex
	requests []copyRequest
}

func (f *copyFixture) snapshot() []copyRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

func (f *copyFixture) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = nil
}

func copyFixtureRepository(id int, name, permission string) map[string]any {
	base := permission
	if strings.HasPrefix(permission, "custom-") {
		base = "pull"
	}
	repo := fixtureRepository(id, name, base)
	repo["role_name"] = permission
	return repo
}

func newAccessCopyFixtureServer(t *testing.T) (*httptest.Server, *copyFixture) {
	t.Helper()
	repo := copyFixtureRepository
	responses := map[string]any{
		"/orgs/example-org/teams/source-team/repos": []any{
			repo(1, "direct-only", "pull"), repo(9, "direct-equal", "pull"),
			repo(2, "masked", "admin"), repo(3, "conflict", "pull"),
			repo(4, "inherited-equal", "pull"), repo(5, "inherited-stronger", "pull"),
			repo(10, "inherited-weaker", "push"), repo(6, "custom", "custom-reader"),
			repo(11, "custom-conflict", "custom-reader"), repo(7, "missing", "push"),
		},
		"/orgs/example-org/teams/source-parent/repos": []any{
			repo(2, "masked", "admin"), repo(4, "inherited-equal", "pull"),
			repo(5, "inherited-stronger", "pull"), repo(10, "inherited-weaker", "push"),
			repo(6, "custom", "custom-reader"), repo(11, "custom-conflict", "custom-reader"),
			repo(7, "missing", "push"),
		},
		"/orgs/example-org/teams/destination-team/repos": []any{
			repo(9, "direct-equal", "pull"), repo(2, "masked", "admin"), repo(3, "conflict", "push"),
			repo(4, "inherited-equal", "pull"), repo(5, "inherited-stronger", "admin"),
			repo(10, "inherited-weaker", "pull"), repo(11, "custom-conflict", "custom-writer"),
			repo(8, "destination-only", "push"),
		},
		"/orgs/example-org/teams/destination-parent/repos": []any{
			repo(9, "direct-equal", "pull"), repo(2, "masked", "admin"),
			repo(4, "inherited-equal", "pull"), repo(5, "inherited-stronger", "admin"),
			repo(10, "inherited-weaker", "pull"), repo(11, "custom-conflict", "custom-writer"),
		},
		"/orgs/example-org/teams/source-team":      map[string]any{"slug": "source-team", "parent": map[string]string{"slug": "source-parent"}},
		"/orgs/example-org/teams/destination-team": map[string]any{"slug": "destination-team", "parent": map[string]string{"slug": "destination-parent"}},
		"/orgs/example-org/teams/empty-team/repos": []any{},
		"/orgs/example-org/teams/empty-team":       map[string]any{"slug": "empty-team"},
		"/orgs/example-org/teams/source-failure/repos": []any{
			repo(12, "alpha", "pull"), repo(13, "missing", "push"),
			repo(14, "failure", "admin"), repo(15, "later", "pull"),
		},
		"/orgs/example-org/teams/source-failure":               map[string]any{"slug": "source-failure"},
		"/orgs/example-org/teams/error-destination/repos":      []any{},
		"/orgs/example-org/teams/error-destination":            map[string]any{"slug": "error-destination"},
		"/orgs/example-org/teams/error-destination-sync/repos": []any{repo(16, "extra", "push")},
		"/orgs/example-org/teams/error-destination-sync":       map[string]any{"slug": "error-destination-sync"},
		"/repos/example-org/direct-equal/teams": []any{
			map[string]string{"slug": "destination-parent", "permission": "pull"},
		},
		"/repos/example-org/masked/teams": []any{
			map[string]string{"slug": "source-parent", "permission": "admin"},
			map[string]string{"slug": "destination-parent", "permission": "admin"},
			map[string]string{"slug": "source-team", "permission": "pull"},
		},
		"/repos/example-org/inherited-equal/teams": []any{
			map[string]string{"slug": "source-parent", "permission": "pull"},
			map[string]string{"slug": "destination-parent", "permission": "pull"},
		},
		"/repos/example-org/inherited-stronger/teams": []any{
			map[string]string{"slug": "source-parent", "permission": "pull"},
			map[string]string{"slug": "destination-parent", "permission": "admin"},
		},
		"/repos/example-org/inherited-weaker/teams": []any{
			map[string]string{"slug": "source-parent", "permission": "push"},
			map[string]string{"slug": "destination-parent", "permission": "pull"},
		},
		"/repos/example-org/custom-conflict/teams": []any{
			map[string]string{"slug": "source-parent", "permission": "custom-reader"},
			map[string]string{"slug": "destination-parent", "permission": "custom-writer"},
		},
		"/repos/example-org/custom/teams":  []any{map[string]string{"slug": "source-parent", "permission": "custom-reader"}},
		"/repos/example-org/missing/teams": []any{map[string]string{"slug": "source-parent", "permission": "push"}},
	}
	fixture := &copyFixture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := copyRequest{method: r.Method, path: r.URL.Path}
		if r.Method == http.MethodPut {
			var body struct {
				Permission string `json:"permission"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode grant: %v", err)
				http.Error(w, "bad grant", http.StatusBadRequest)
				return
			}
			request.permission = body.Permission
		}
		fixture.mu.Lock()
		fixture.requests = append(fixture.requests, request)
		fixture.mu.Unlock()
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected GitHub API method: %s %s", r.Method, r.URL.Path)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if response, ok := responses[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Errorf("failed to encode GitHub API response: %v", err)
			}
			return
		}
		if strings.HasPrefix(r.URL.Path, "/repos/example-org/") && !strings.HasSuffix(r.URL.Path, "/teams") {
			name := strings.TrimPrefix(r.URL.Path, "/repos/example-org/")
			if name == "missing" {
				http.NotFound(w, r)
				return
			}
			if name == "failure" {
				http.Error(w, "permission denied", http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(map[string]any{"name": name, "owner": map[string]string{"login": "example-org"}}); err != nil {
				t.Errorf("failed to encode repository: %v", err)
			}
			return
		}
		t.Errorf("unexpected GitHub API request: %s", r.URL.String())
		http.NotFound(w, r)
	}))
	return server, fixture
}

func TestAccessCopyModesMatchDryRunAndApply(t *testing.T) {
	const destination = "example-org/destination-team"
	add := func(name, permission, current string) gh.SyncChange {
		return gh.SyncChange{Destination: destination, Action: gh.SyncChangeAdd, Target: name, Permission: permission, Current: current}
	}
	update := func(name, permission, current string) gh.SyncChange {
		return gh.SyncChange{Destination: destination, Action: gh.SyncChangeUpdate, Target: name, Permission: permission, Current: current}
	}
	tests := []struct {
		name         string
		flag         string
		force        string
		wantChanges  []gh.SyncChange
		wantWarnings []string
	}{
		{
			name: "direct",
			wantChanges: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"),
			},
			wantWarnings: []string{"repository conflict would downgrade direct permission from push to pull"},
		},
		{
			name:  "direct with short force flag",
			force: "-f",
			wantChanges: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"), update("conflict", "pull", "push"),
			},
		},
		{
			name: "include inherited",
			flag: "--include-inherited",
			wantChanges: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"),
				add("inherited-weaker", "push", "pull"), add("custom", "custom-reader", ""),
			},
			wantWarnings: []string{"repository conflict would downgrade", "repository custom-conflict has incomparable", "repository missing is unavailable"},
		},
		{
			name:  "include inherited with force",
			flag:  "--include-inherited",
			force: "--force",
			wantChanges: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"), update("conflict", "pull", "push"),
				add("inherited-weaker", "push", "pull"), add("custom", "custom-reader", ""),
				add("custom-conflict", "custom-reader", "custom-writer"),
			},
			wantWarnings: []string{"repository missing is unavailable"},
		},
		{
			name: "effective",
			flag: "--effective",
			wantChanges: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "admin", "admin"),
				add("inherited-stronger", "pull", "admin"), add("inherited-weaker", "push", "pull"),
				add("custom", "custom-reader", ""),
			},
			wantWarnings: []string{"repository conflict would downgrade", "repository inherited-stronger inherits destination parent permission admin", "repository custom-conflict has incomparable", "repository missing is unavailable"},
		},
		{
			name:  "effective with force",
			flag:  "--effective",
			force: "--force",
			wantChanges: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "admin", "admin"), update("conflict", "pull", "push"),
				add("inherited-stronger", "pull", "admin"), add("inherited-weaker", "push", "pull"),
				add("custom", "custom-reader", ""), add("custom-conflict", "custom-reader", "custom-writer"),
			},
			wantWarnings: []string{"repository inherited-stronger inherits destination parent permission admin", "repository custom-conflict has incomparable", "effective access may differ", "repository missing is unavailable"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, fixture := newAccessCopyFixtureServer(t)
			defer server.Close()
			t.Setenv("GH_CONFIG_DIR", t.TempDir())
			t.Setenv("GH_HOST", "github.com")
			t.Setenv("GH_TOKEN", "test-token")
			t.Setenv("GITHUB_API_URL", server.URL)

			args := []string{"example-org/source-team", destination}
			if tt.flag != "" {
				args = append(args, tt.flag)
			}
			if tt.force != "" {
				args = append(args, tt.force)
			}
			dryArgs := append(slices.Clone(args), "--dryrun", "--format", "json")
			stdout, stderr, status := runAccessCopy(t, dryArgs...)
			if status != 0 {
				t.Fatalf("dry run failed (%d): %s", status, stderr)
			}
			var got []gh.SyncChange
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("invalid dry-run JSON %q: %v", stdout, err)
			}
			if !reflect.DeepEqual(got, tt.wantChanges) {
				t.Fatalf("dry-run changes = %#v, want %#v", got, tt.wantChanges)
			}
			for _, warning := range tt.wantWarnings {
				if !strings.Contains(stderr, warning) {
					t.Errorf("dry-run stderr = %q, want %q", stderr, warning)
				}
			}
			for _, request := range fixture.snapshot() {
				if request.method == http.MethodPut {
					t.Fatalf("dry run applied a grant: %#v", request)
				}
			}

			fixture.reset()
			_, stderr, status = runAccessCopy(t, args...)
			if status != 0 {
				t.Fatalf("copy failed (%d): %s", status, stderr)
			}
			for _, warning := range tt.wantWarnings {
				if !strings.Contains(stderr, warning) {
					t.Errorf("copy stderr = %q, want %q", stderr, warning)
				}
			}
			var grants []copyRequest
			for _, request := range fixture.snapshot() {
				if request.method == http.MethodPut {
					grants = append(grants, request)
				}
			}
			var wantGrants []copyRequest
			for _, change := range tt.wantChanges {
				wantGrants = append(wantGrants, copyRequest{
					method: http.MethodPut, path: "/orgs/example-org/teams/destination-team/repos/example-org/" + change.Target,
					permission: change.Permission,
				})
			}
			if !reflect.DeepEqual(grants, wantGrants) {
				t.Errorf("applied grants = %#v, want %#v", grants, wantGrants)
			}
		})
	}
}

func TestAccessCopyRejectsConflictingModes(t *testing.T) {
	server, fixture := newAccessCopyFixtureServer(t)
	defer server.Close()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)
	for _, flags := range [][]string{{"--include-inherited", "--effective"}, {"--include-inherited", "--effective", "--force"}} {
		t.Run(strings.Join(flags, "-"), func(t *testing.T) {
			fixture.reset()
			args := append([]string{"example-org/source-team", "example-org/destination-team"}, flags...)
			_, _, status := runAccessCopy(t, args...)
			if status == 0 {
				t.Fatalf("flags %q unexpectedly succeeded", flags)
			}
			if requests := fixture.snapshot(); len(requests) != 0 {
				t.Fatalf("flags %q caused API requests: %#v", flags, requests)
			}
		})
	}
}

func TestAccessCopyDryRunWithoutChanges(t *testing.T) {
	server, fixture := newAccessCopyFixtureServer(t)
	defer server.Close()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	stdout, stderr, status := runAccessCopy(t, "example-org/empty-team", "example-org/destination-team", "--dryrun")
	if status != 0 || stdout != "No changes.\n" {
		t.Fatalf("dry run = %q, %q, %d; want No changes", stdout, stderr, status)
	}
	stdout, stderr, status = runAccessCopy(t, "example-org/empty-team", "example-org/destination-team", "--dryrun", "--format", "json")
	if status != 0 || strings.TrimSpace(stdout) != "[]" {
		t.Fatalf("JSON dry run = %q, %q, %d; want []", stdout, stderr, status)
	}
	for _, request := range fixture.snapshot() {
		if request.method == http.MethodPut {
			t.Fatalf("dry run applied a grant: %#v", request)
		}
	}
}

func TestAccessCopyAppliesOnlyValidatedPrefixOnLookupError(t *testing.T) {
	server, fixture := newAccessCopyFixtureServer(t)
	defer server.Close()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	args := []string{"example-org/source-failure", "example-org/error-destination"}
	_, stderr, status := runAccessCopy(t, append(slices.Clone(args), "--dryrun")...)
	if status == 0 || !strings.Contains(stderr, "repository missing is unavailable") ||
		!strings.Contains(stderr, "failed to check destination repository example-org/failure") {
		t.Fatalf("dry-run error = %q, status %d", stderr, status)
	}
	for _, request := range fixture.snapshot() {
		if request.method == http.MethodPut {
			t.Fatalf("failed dry run applied a grant: %#v", request)
		}
	}

	fixture.reset()
	_, stderr, status = runAccessCopy(t, args...)
	if status == 0 || !strings.Contains(stderr, "repository missing is unavailable") ||
		!strings.Contains(stderr, "failed to check destination repository example-org/failure") {
		t.Fatalf("copy error = %q, status %d", stderr, status)
	}
	var grants []copyRequest
	for _, request := range fixture.snapshot() {
		if request.method == http.MethodPut {
			grants = append(grants, request)
		}
		if request.path == "/repos/example-org/later" {
			t.Fatal("repository validation continued after a non-404 error")
		}
	}
	want := []copyRequest{{
		method: http.MethodPut, path: "/orgs/example-org/teams/error-destination/repos/example-org/alpha", permission: "pull",
	}}
	if !reflect.DeepEqual(grants, want) {
		t.Errorf("applied grants = %#v, want %#v", grants, want)
	}
}
