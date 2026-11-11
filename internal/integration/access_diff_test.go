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
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	teamcmd "github.com/srz-zumix/gh-team-kit/cmd"
	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
)

func TestAccessDiffProcess(t *testing.T) {
	if os.Getenv("GH_TEAM_KIT_TEST_ACCESS_DIFF") != "1" {
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
	cmd.SetArgs(append([]string{"access", "diff"}, os.Args[separator+1:]...))
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func runAccessDiff(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testArgs := []string{"-test.run=^TestAccessDiffProcess$", "--", "--color=never"}
	testArgs = append(testArgs, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], testArgs...)
	cmd.Env = append(os.Environ(), "GH_TEAM_KIT_TEST_ACCESS_DIFF=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("access diff process timed out: %v", ctx.Err())
	}
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	t.Fatalf("failed to run access diff process: %v", err)
	return "", "", -1
}

func fixtureRepository(id int, name, permission string) map[string]any {
	permissions := map[string]bool{"pull": true}
	switch permission {
	case "pull":
	case "push":
		permissions["triage"] = true
		permissions["push"] = true
	case "admin":
		permissions["triage"] = true
		permissions["push"] = true
		permissions["maintain"] = true
		permissions["admin"] = true
	default:
		panic("unsupported fixture permission: " + permission)
	}
	return map[string]any{
		"id":          id,
		"name":        name,
		"full_name":   "example-org/" + name,
		"owner":       map[string]string{"login": "example-org"},
		"permissions": permissions,
	}
}

func newAccessDiffFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	inheritedA := fixtureRepository(1, "inherited-a", "pull")
	inheritedB := fixtureRepository(2, "inherited-b", "pull")
	shared := fixtureRepository(3, "shared", "admin")
	equalDirect := fixtureRepository(4, "equal-direct", "admin")
	directA := fixtureRepository(5, "direct-a", "push")
	directB := fixtureRepository(6, "direct-b", "push")
	strongerA := fixtureRepository(7, "stronger-a", "push")
	parentStrongerA := fixtureRepository(7, "stronger-a", "pull")
	rootOnly := fixtureRepository(8, "root-only", "pull")
	customRole := fixtureRepository(9, "custom-role", "admin")

	responses := map[string]any{
		"/orgs/example-org/teams/team-a/repos":   []any{inheritedA, shared, equalDirect, directA, strongerA, customRole},
		"/orgs/example-org/teams/team-b/repos":   []any{inheritedB, shared, equalDirect, directB, customRole},
		"/orgs/example-org/teams/parent-a/repos": []any{inheritedA, shared, equalDirect, parentStrongerA, customRole},
		"/orgs/example-org/teams/parent-b/repos": []any{inheritedB, shared, equalDirect, customRole},
		"/orgs/example-org/teams/root-team/repos": []any{
			rootOnly,
		},
		"/orgs/example-org/teams/fail-team/repos": []any{},
		"/orgs/example-org/teams/team-a":          map[string]any{"slug": "team-a", "parent": map[string]string{"slug": "parent-a"}},
		"/orgs/example-org/teams/team-b":          map[string]any{"slug": "team-b", "parent": map[string]string{"slug": "parent-b"}},
		"/orgs/example-org/teams/root-team":       map[string]any{"slug": "root-team", "parent": nil},
		"/orgs/example-org/teams/fail-team":       map[string]any{"slug": "fail-team", "parent": map[string]string{"slug": "fail-parent"}},
		"/repos/example-org/inherited-a/teams":    []any{map[string]string{"slug": "parent-a", "permission": "pull"}},
		"/repos/example-org/inherited-b/teams":    []any{map[string]string{"slug": "parent-b", "permission": "pull"}},
		"/repos/example-org/shared/teams": []any{
			map[string]string{"slug": "parent-a", "permission": "admin"},
			map[string]string{"slug": "parent-b", "permission": "admin"},
			map[string]string{"slug": "team-a", "permission": "pull"},
			map[string]string{"slug": "team-b", "permission": "push"},
		},
		"/repos/example-org/equal-direct/teams": []any{
			map[string]string{"slug": "parent-a", "permission": "admin"},
			map[string]string{"slug": "parent-b", "permission": "admin"},
			map[string]any{"slug": "team-a", "permission": "pull", "permissions": map[string]bool{"pull": true}},
			map[string]string{"slug": "team-b", "permission": "read"},
		},
		"/repos/example-org/custom-role/teams": []any{
			map[string]string{"slug": "parent-a", "permission": "admin"},
			map[string]string{"slug": "parent-b", "permission": "admin"},
			map[string]any{"slug": "team-a", "permission": "security-reviewer", "permissions": map[string]bool{"pull": true}},
			map[string]string{"slug": "team-b", "permission": "push"},
		},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected GitHub API method: %s %s", r.Method, r.URL.Path)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path == "/orgs/example-org/teams/fail-parent/repos" {
			http.Error(w, "upstream failure", http.StatusBadGateway)
			return
		}
		response, ok := responses[r.URL.Path]
		if !ok {
			t.Errorf("unexpected GitHub API request: %s", r.URL.String())
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("failed to encode GitHub API response: %v", err)
		}
	}))
}

func TestAccessDiffInheritedPermissions(t *testing.T) {
	server := newAccessDiffFixtureServer(t)
	defer server.Close()
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("GH_HOST", "github.com")
	t.Setenv("GH_TOKEN", "test-token")
	t.Setenv("GITHUB_API_URL", server.URL)

	const inheritedABlock = "diff --gh team-kit access diff team-a team-b example-org/inherited-a\n" +
		"--- example-org/inherited-a team-a\n+++ /dev/null\n- pull\n"
	const inheritedBBlock = "diff --gh team-kit access diff team-a team-b example-org/inherited-b\n" +
		"--- /dev/null\n+++ example-org/inherited-b team-b\n+ pull\n"
	const directSharedBlock = "diff --gh team-kit access diff team-a team-b example-org/shared --no-inherit\n" +
		"--- example-org/shared team-a\n+++ example-org/shared team-b\n- pull\n+ push\n"
	const directABlock = "diff --gh team-kit access diff team-a team-b example-org/direct-a --no-inherit\n" +
		"--- example-org/direct-a team-a\n+++ /dev/null\n- push\n"
	const directBBlock = "diff --gh team-kit access diff team-a team-b example-org/direct-b --no-inherit\n" +
		"--- /dev/null\n+++ example-org/direct-b team-b\n+ push\n"
	const strongerABlock = "diff --gh team-kit access diff team-a team-b example-org/stronger-a --no-inherit\n" +
		"--- example-org/stronger-a team-a\n+++ /dev/null\n- push\n"
	const customRoleBlock = "diff --gh team-kit access diff team-a team-b example-org/custom-role --no-inherit\n" +
		"--- example-org/custom-role team-a\n+++ example-org/custom-role team-b\n- security-reviewer\n+ push\n"
	const noDifferencesLog = `level=INFO msg="No differences found."`

	tests := []struct {
		name            string
		args            []string
		wantOutput      []string
		avoidOutput     []string
		wantEmptyOutput bool
		wantExit        int
		wantError       string
		wantLog         string
	}{
		{
			name:        "default includes inherited permissions on both sides",
			args:        []string{"example-org/team-a", "example-org/team-b"},
			wantOutput:  []string{inheritedABlock, inheritedBBlock, "example-org/direct-a", "example-org/direct-b", "example-org/stronger-a"},
			avoidOutput: []string{"--no-inherit", "example-org/shared", "example-org/equal-direct", "example-org/custom-role"},
		},
		{
			name:        "no-inherit compares direct permissions on both sides",
			args:        []string{"example-org/team-a", "example-org/team-b", "--no-inherit"},
			wantOutput:  []string{directSharedBlock, directABlock, directBBlock, strongerABlock, customRoleBlock},
			avoidOutput: []string{"example-org/inherited-a", "example-org/inherited-b", "example-org/equal-direct"},
		},
		{
			name:        "explicit false preserves effective comparison",
			args:        []string{"example-org/team-a", "example-org/team-b", "inherited-a", "--no-inherit=false", "--exit-code"},
			wantOutput:  []string{inheritedABlock},
			avoidOutput: []string{"--no-inherit", "example-org/direct-a"},
			wantExit:    1,
		},
		{
			name:        "filtered direct difference controls exit code",
			args:        []string{"example-org/team-a", "example-org/team-b", "example-org/shared", "--no-inherit", "--exit-code"},
			wantOutput:  []string{directSharedBlock},
			avoidOutput: []string{"example-org/direct-a", "example-org/inherited-a"},
			wantExit:    1,
		},
		{
			name:            "filtered inherited difference is ignored",
			args:            []string{"example-org/team-a", "example-org/team-b", "inherited-a", "--no-inherit", "--exit-code"},
			wantEmptyOutput: true,
			wantLog:         noDifferencesLog,
		},
		{
			name:            "matching direct permissions are ignored",
			args:            []string{"example-org/team-a", "example-org/team-b", "equal-direct", "--no-inherit", "--exit-code"},
			wantEmptyOutput: true,
			wantLog:         noDifferencesLog,
		},
		{
			name:            "matching effective permissions",
			args:            []string{"example-org/team-a", "example-org/team-a"},
			wantEmptyOutput: true,
			wantLog:         noDifferencesLog,
		},
		{
			name:            "matching permissions with forced color",
			args:            []string{"example-org/team-a", "example-org/team-a", "--color", "always"},
			wantEmptyOutput: true,
			wantLog:         noDifferencesLog,
		},
		{
			name:            "info log respects the log level",
			args:            []string{"example-org/team-a", "example-org/team-a", "--log-level", "warn"},
			wantEmptyOutput: true,
		},
		{
			name:       "team without a parent retains its direct permissions",
			args:       []string{"example-org/team-a", "example-org/root-team", "root-only", "--no-inherit", "--exit-code"},
			wantOutput: []string{"diff --gh team-kit access diff team-a root-team example-org/root-only --no-inherit\n--- /dev/null\n+++ example-org/root-only root-team\n+ pull\n"},
			wantExit:   1,
		},
		{
			name:      "source parent lookup failure",
			args:      []string{"example-org/fail-team", "example-org/team-b", "--no-inherit"},
			wantExit:  1,
			wantError: "failed to fetch repositories for team fail-team",
		},
		{
			name:      "destination parent lookup failure",
			args:      []string{"example-org/team-a", "example-org/fail-team", "--no-inherit"},
			wantExit:  1,
			wantError: "failed to fetch repositories for team fail-team",
		},
		{
			name:      "JSON rendering failure",
			args:      []string{"example-org/team-a", "example-org/team-a", "--format", "json", "--jq", ".["},
			wantExit:  1,
			wantError: "failed to render team repository differences",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode := runAccessDiff(t, tt.args...)
			if exitCode != tt.wantExit {
				t.Fatalf("exit code = %d, want %d; stdout = %q; stderr = %q", exitCode, tt.wantExit, stdout, stderr)
			}
			if tt.wantEmptyOutput && stdout != "" {
				t.Errorf("stdout = %q, want no output", stdout)
			}
			for _, want := range tt.wantOutput {
				if !strings.Contains(stdout, want) {
					t.Errorf("stdout %q does not contain %q", stdout, want)
				}
			}
			for _, unwanted := range tt.avoidOutput {
				if strings.Contains(stdout, unwanted) {
					t.Errorf("stdout %q unexpectedly contains %q", stdout, unwanted)
				}
			}
			if tt.wantError != "" && !strings.Contains(stderr, tt.wantError) {
				t.Errorf("stderr %q does not contain %q", stderr, tt.wantError)
			}
			if tt.wantLog != "" && !strings.Contains(stderr, tt.wantLog) {
				t.Errorf("stderr %q does not contain log message %q", stderr, tt.wantLog)
			}
			if tt.wantLog != "" && (!strings.Contains(stderr, "left="+tt.args[0]) || !strings.Contains(stderr, "right="+tt.args[1])) {
				t.Errorf("stderr %q does not contain both compared teams", stderr)
			}
			if tt.wantError != "" && strings.Contains(stderr, noDifferencesLog) {
				t.Errorf("unexpected no-differences log after an error: %s", stderr)
			}
			if tt.wantError == "" && tt.wantLog == "" && stderr != "" {
				t.Errorf("unexpected stderr: %s", stderr)
			}
		})
	}

	t.Run("JSON output contains direct permissions", func(t *testing.T) {
		stdout, stderr, exitCode := runAccessDiff(t, "example-org/team-a", "example-org/team-b", "shared", "--no-inherit", "--format", "json")
		if exitCode != 0 {
			t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout, stderr)
		}
		if stderr != "" {
			t.Errorf("unexpected stderr: %s", stderr)
		}
		type repo struct {
			Name     string `json:"name"`
			RoleName string `json:"role_name"`
		}
		var differences []struct {
			Left  *repo
			Right *repo
		}
		if err := json.Unmarshal([]byte(stdout), &differences); err != nil {
			t.Fatalf("invalid JSON output %q: %v", stdout, err)
		}
		if len(differences) != 1 || differences[0].Left == nil || differences[0].Right == nil {
			t.Fatalf("unexpected JSON differences: %s", stdout)
		}
		if left, right := differences[0].Left, differences[0].Right; left.Name != "shared" || right.Name != "shared" || left.RoleName != "pull" || right.RoleName != "push" {
			t.Errorf("unexpected direct permissions in JSON output: %s", stdout)
		}
	})

	for _, tt := range []struct {
		name       string
		flags      []string
		wantOutput string
	}{
		{name: "JSON", wantOutput: "[]\n"},
		{name: "jq", flags: []string{"--jq", "length"}, wantOutput: "0\n"},
		{name: "template", flags: []string{"--template", "{{len .}}"}, wantOutput: "0"},
	} {
		t.Run("no differences preserve "+tt.name+" output", func(t *testing.T) {
			args := []string{"example-org/team-a", "example-org/team-b", "equal-direct", "--no-inherit", "--format", "json", "--exit-code"}
			args = append(args, tt.flags...)
			stdout, stderr, exitCode := runAccessDiff(t, args...)
			if exitCode != 0 {
				t.Fatalf("exit code = %d; stdout = %q; stderr = %q", exitCode, stdout, stderr)
			}
			if stdout != tt.wantOutput {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantOutput)
			}
			if !strings.Contains(stderr, noDifferencesLog) {
				t.Errorf("stderr %q does not contain the no-differences log", stderr)
			}
		})
	}
}
