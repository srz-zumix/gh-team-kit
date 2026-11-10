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
	"testing"
	"time"

	"github.com/spf13/cobra"
	teamcmd "github.com/srz-zumix/gh-team-kit/cmd"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

func TestAccessSyncProcess(t *testing.T) {
	if os.Getenv("GH_TEAM_KIT_TEST_ACCESS_SYNC") != "1" {
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
	cmd.SetArgs(append([]string{"access", "sync"}, os.Args[separator+1:]...))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func runAccessSync(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	testArgs := append([]string{"-test.run=^TestAccessSyncProcess$", "--"}, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], testArgs...)
	cmd.Env = append(os.Environ(), "GH_TEAM_KIT_TEST_ACCESS_SYNC=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("access sync process timed out: %v", ctx.Err())
	}
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	t.Fatalf("failed to run access sync process: %v", err)
	return "", "", -1
}

func TestAccessSyncModesDryRun(t *testing.T) {
	const destination = "example-org/destination-team"
	add := func(name, permission, current string) gh.SyncChange {
		return gh.SyncChange{Destination: destination, Action: gh.SyncChangeAdd, Target: name, Permission: permission, Current: current}
	}
	update := func(name, permission, current string) gh.SyncChange {
		return gh.SyncChange{Destination: destination, Action: gh.SyncChangeUpdate, Target: name, Permission: permission, Current: current}
	}
	remove := gh.SyncChange{Destination: destination, Action: gh.SyncChangeRemove, Target: "destination-only", Current: "push"}
	tests := []struct {
		name         string
		flag         string
		force        string
		want         []gh.SyncChange
		wantWarnings []string
	}{
		{
			name: "direct",
			flag: "default",
			want: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"),
			},
			wantWarnings: []string{"repository conflict would downgrade", "repository destination-only cannot safely remove"},
		},
		{
			name:  "direct with short force flag",
			flag:  "default",
			force: "-f",
			want: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"), update("conflict", "pull", "push"), remove,
			},
		},
		{
			name: "include inherited",
			flag: "--include-inherited",
			want: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"),
				add("inherited-weaker", "push", "pull"), add("custom", "custom-reader", ""),
			},
			wantWarnings: []string{"repository conflict would downgrade", "repository custom-conflict has incomparable", "repository destination-only cannot safely remove", "repository missing is unavailable"},
		},
		{
			name:  "include inherited with force",
			flag:  "--include-inherited",
			force: "--force",
			want: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "pull", "admin"), update("conflict", "pull", "push"),
				add("inherited-weaker", "push", "pull"), add("custom", "custom-reader", ""),
				add("custom-conflict", "custom-reader", "custom-writer"), remove,
			},
			wantWarnings: []string{"repository custom-conflict has incomparable", "effective access may differ", "repository missing is unavailable"},
		},
		{
			name: "effective",
			flag: "--effective",
			want: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "admin", "admin"),
				add("inherited-stronger", "pull", "admin"), add("inherited-weaker", "push", "pull"),
				add("custom", "custom-reader", ""),
			},
			wantWarnings: []string{"repository conflict would downgrade", "repository inherited-stronger inherits destination parent permission admin", "repository custom-conflict has incomparable", "repository destination-only cannot safely remove", "repository missing is unavailable"},
		},
		{
			name:  "effective with force",
			flag:  "--effective",
			force: "--force",
			want: []gh.SyncChange{
				add("direct-only", "pull", ""), add("direct-equal", "pull", "pull"),
				add("masked", "admin", "admin"), update("conflict", "pull", "push"),
				add("inherited-stronger", "pull", "admin"), add("inherited-weaker", "push", "pull"),
				add("custom", "custom-reader", ""), add("custom-conflict", "custom-reader", "custom-writer"), remove,
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
			if tt.flag != "default" {
				args = append(args, tt.flag)
			}
			if tt.force != "" {
				args = append(args, tt.force)
			}
			stdout, stderr, status := runAccessSync(t, append(args, "--dryrun", "--format", "json")...)
			if status != 0 {
				t.Fatalf("dry run failed (%d): %s", status, stderr)
			}
			var changes []gh.SyncChange
			if err := json.Unmarshal([]byte(stdout), &changes); err != nil {
				t.Fatalf("invalid JSON %q: %v", stdout, err)
			}
			if !reflect.DeepEqual(changes, tt.want) {
				t.Fatalf("changes = %#v, want %#v", changes, tt.want)
			}
			for _, warning := range tt.wantWarnings {
				if !strings.Contains(stderr, warning) {
					t.Errorf("stderr = %q, want %q", stderr, warning)
				}
			}
			for _, request := range fixture.snapshot() {
				if request.method != http.MethodGet {
					t.Fatalf("dry run mutated access: %#v", request)
				}
			}
		})
	}
}

func TestAccessSyncRejectsConflictingModes(t *testing.T) {
	server, fixture := newAccessCopyFixtureServer(t)
	defer server.Close()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)
	_, _, status := runAccessSync(t, "example-org/source-team", "example-org/destination-team", "--include-inherited", "--effective")
	if status == 0 || len(fixture.snapshot()) != 0 {
		t.Fatalf("conflicting flags: status %d, requests %#v", status, fixture.snapshot())
	}
}

func TestAccessSyncApplyMatchesDryRun(t *testing.T) {
	for _, mode := range []string{"", "--include-inherited", "--effective"} {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/force=%t", mode, force), func(t *testing.T) {
				fixture := &copyFixture{}
				destination := map[string]string{"extra": "push"}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					fixture.mu.Lock()
					defer fixture.mu.Unlock()
					request := copyRequest{method: r.Method, path: r.URL.Path}
					switch r.Method {
					case http.MethodPut:
						var body struct {
							Permission string `json:"permission"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("decode grant: %v", err)
							http.Error(w, "bad grant", http.StatusBadRequest)
							return
						}
						request.permission = body.Permission
						destination["alpha"] = body.Permission
					case http.MethodDelete:
						delete(destination, "extra")
					}
					fixture.requests = append(fixture.requests, request)
					if r.Method == http.MethodPut || r.Method == http.MethodDelete {
						w.WriteHeader(http.StatusNoContent)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					var response any
					switch r.URL.Path {
					case "/orgs/example-org/teams/source/repos":
						response = []any{copyFixtureRepository(1, "alpha", "pull")}
					case "/orgs/example-org/teams/source":
						response = map[string]string{"slug": "source"}
					case "/orgs/example-org/teams/destination":
						response = map[string]string{"slug": "destination"}
					case "/orgs/example-org/teams/destination/repos":
						repos := []any{}
						for name, permission := range destination {
							repos = append(repos, copyFixtureRepository(2, name, permission))
						}
						response = repos
					case "/repos/example-org/alpha":
						response = map[string]string{"name": "alpha"}
					default:
						t.Errorf("unexpected API request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
						return
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Errorf("encode response: %v", err)
					}
				}))
				defer server.Close()
				t.Setenv("GH_CONFIG_DIR", t.TempDir())
				t.Setenv("GH_HOST", "github.com")
				t.Setenv("GH_TOKEN", "test-token")
				t.Setenv("GITHUB_API_URL", server.URL)
				args := []string{"example-org/source", "example-org/destination"}
				if mode != "" {
					args = append(args, mode)
				}
				if force {
					args = append(args, "--force")
				}
				stdout, stderr, status := runAccessSync(t, append(slices.Clone(args), "--dryrun", "--format", "json")...)
				if status != 0 {
					t.Fatalf("dry run failed (%d): %s", status, stderr)
				}
				var changes []gh.SyncChange
				if err := json.Unmarshal([]byte(stdout), &changes); err != nil {
					t.Fatal(err)
				}
				want := []gh.SyncChange{
					{Destination: "example-org/destination", Action: gh.SyncChangeAdd, Target: "alpha", Permission: "pull"},
				}
				if force {
					want = append(want, gh.SyncChange{Destination: "example-org/destination", Action: gh.SyncChangeRemove, Target: "extra", Current: "push"})
				} else if !strings.Contains(stderr, "repository extra cannot safely remove") {
					t.Errorf("dry-run warning missing: %q", stderr)
				}
				if !reflect.DeepEqual(changes, want) {
					t.Fatalf("dry-run changes = %#v, want %#v", changes, want)
				}
				fixture.reset()
				_, stderr, status = runAccessSync(t, args...)
				if status != 0 {
					t.Fatalf("sync failed (%d): %s", status, stderr)
				}
				if !force && (!strings.Contains(stderr, "repository extra cannot safely remove") ||
					strings.Contains(stderr, "repository extra remains accessible after synchronization")) {
					t.Errorf("unexpected skipped-repository warnings: %q", stderr)
				}
				var mutations []copyRequest
				for _, request := range fixture.snapshot() {
					if request.method != http.MethodGet {
						mutations = append(mutations, request)
					}
				}

				wantMutations := []copyRequest{
					{method: http.MethodPut, path: "/orgs/example-org/teams/destination/repos/example-org/alpha", permission: "pull"},
				}
				if force {
					wantMutations = append(wantMutations, copyRequest{method: http.MethodDelete, path: "/orgs/example-org/teams/destination/repos/example-org/extra"})
				}
				if !reflect.DeepEqual(mutations, wantMutations) {
					t.Errorf("mutations = %#v, want %#v", mutations, wantMutations)
				}
			})
		}
	}
}

func TestAccessSyncAppliesOnlyValidatedGrantsOnLookupError(t *testing.T) {
	for _, mode := range []string{"", "--include-inherited", "--effective"} {
		t.Run(mode, func(t *testing.T) {
			server, fixture := newAccessCopyFixtureServer(t)
			defer server.Close()
			t.Setenv("GH_CONFIG_DIR", t.TempDir())
			t.Setenv("GH_HOST", "github.com")
			t.Setenv("GH_TOKEN", "test-token")
			t.Setenv("GITHUB_API_URL", server.URL)

			args := []string{"example-org/source-failure", "example-org/error-destination-sync"}
			if mode != "" {
				args = append(args, mode)
			}
			_, stderr, status := runAccessSync(t, append(slices.Clone(args), "--dryrun")...)
			if status == 0 || !strings.Contains(stderr, "repository missing is unavailable") ||
				!strings.Contains(stderr, "failed to check destination repository example-org/failure") {
				t.Fatalf("dry-run error = %q, status %d", stderr, status)
			}
			for _, request := range fixture.snapshot() {
				if request.method != http.MethodGet {
					t.Fatalf("failed dry run mutated access: %#v", request)
				}
			}
			fixture.reset()
			_, stderr, status = runAccessSync(t, args...)
			if status == 0 || !strings.Contains(stderr, "failed to check destination repository example-org/failure") {
				t.Fatalf("sync error = %q, status %d", stderr, status)
			}
			var mutations []copyRequest
			for _, request := range fixture.snapshot() {
				if request.method != http.MethodGet {
					mutations = append(mutations, request)
				}
			}
			want := []copyRequest{{
				method:     http.MethodPut,
				path:       "/orgs/example-org/teams/error-destination-sync/repos/example-org/alpha",
				permission: "pull",
			}}
			if !reflect.DeepEqual(mutations, want) {
				t.Errorf("mutations = %#v, want %#v", mutations, want)
			}
		})
	}
}
