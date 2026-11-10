package teamaccess

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

func TestPlanGrantsStandardPermissionSafety(t *testing.T) {
	for sourceRank, sourcePermission := range gh.PermissionsList {
		for destinationRank, destinationPermission := range gh.PermissionsList {
			for _, force := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s-to-%s/force=%t", destinationPermission, sourcePermission, force), func(t *testing.T) {
					g, transport := newCopyFixtureClient(t, map[string][]copyFixtureResponse{
						"GET /repos/destination-org/sample": fixtureOK(`{"name":"sample"}`),
					})
					_, destination := fixtureTeams(g)
					source := []*github.Repository{testRepository("sample", sourcePermission)}
					current := []*github.Repository{testRepository("sample", destinationPermission)}
					plan, err := planGrants(context.Background(), destination, source, current, current, nil, true, force)
					if err != nil {
						t.Fatal(err)
					}

					wantSkip := !force && sourceRank > destinationRank
					wantChange := sourceRank != destinationRank && !wantSkip
					if got := len(plan.Changes()) == 1; got != wantChange {
						t.Fatalf("changes = %#v, want change %t", plan.Changes(), wantChange)
					}
					if wantChange {
						change := plan.Changes()[0]
						if change.Action != gh.SyncChangeUpdate || change.Permission != sourcePermission || change.Current != destinationPermission {
							t.Errorf("change = %#v", change)
						}
					}
					_, skipped := plan.skipped["sample"]
					if skipped != wantSkip || (len(plan.Warnings()) == 1) != wantSkip {
						t.Errorf("skipped = %t, warnings = %q; want skip %t", skipped, plan.Warnings(), wantSkip)
					}
					if wantSkip && !strings.Contains(plan.Warnings()[0], "use --force to override") {
						t.Errorf("warning does not explain force: %q", plan.Warnings())
					}
					if (len(transport.requests) == 1) != wantChange {
						t.Errorf("repository checks = %#v, want check %t", transport.requests, wantChange)
					}
				})
			}
		}
	}
}

func TestPlanGrantsCustomAndInheritedPermissions(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		direct      string
		effective   string
		sameOrg     bool
		force       bool
		wantAction  gh.SyncChangeAction
		wantSkip    bool
		wantWarning string
	}{
		{name: "new custom role", source: "custom-reviewer", sameOrg: true, wantAction: gh.SyncChangeAdd},
		{name: "custom over standard", source: "custom-reviewer", direct: "pull", effective: "pull", sameOrg: true, wantSkip: true},
		{name: "forced custom over standard", source: "custom-reviewer", direct: "pull", effective: "pull", sameOrg: true, force: true, wantAction: gh.SyncChangeUpdate},
		{name: "standard over custom", source: "admin", direct: "custom-reviewer", effective: "admin", sameOrg: true, wantSkip: true},
		{name: "forced standard over custom", source: "admin", direct: "custom-reviewer", effective: "admin", sameOrg: true, force: true, wantAction: gh.SyncChangeUpdate},
		{name: "different custom roles", source: "custom-reviewer", direct: "custom-editor", effective: "custom-editor", sameOrg: true, wantSkip: true},
		{name: "forced different custom roles", source: "custom-reviewer", direct: "custom-editor", effective: "custom-editor", sameOrg: true, force: true, wantAction: gh.SyncChangeUpdate},
		{name: "same custom role in same organization", source: "custom-reviewer", direct: "custom-reviewer", effective: "custom-reviewer", sameOrg: true},
		{name: "same custom role in different organization", source: "custom-reviewer", direct: "custom-reviewer", effective: "custom-reviewer", wantSkip: true},
		{name: "forced same custom role in different organization", source: "custom-reviewer", direct: "custom-reviewer", effective: "custom-reviewer", force: true},
		{name: "custom over inherited standard", source: "custom-reviewer", effective: "admin", sameOrg: true, wantSkip: true},
		{name: "forced custom over inherited standard", source: "custom-reviewer", effective: "admin", sameOrg: true, force: true, wantAction: gh.SyncChangeAdd},
		{name: "standard over inherited custom", source: "pull", effective: "custom-reviewer", sameOrg: true, wantSkip: true},
		{name: "forced standard over inherited custom", source: "pull", effective: "custom-reviewer", sameOrg: true, force: true, wantAction: gh.SyncChangeAdd},
		{name: "same inherited custom role in same organization", source: "custom-reviewer", effective: "custom-reviewer", sameOrg: true, wantAction: gh.SyncChangeAdd},
		{name: "same inherited custom role in different organization", source: "custom-reviewer", effective: "custom-reviewer", wantSkip: true},
		{name: "forced same inherited custom role in different organization", source: "custom-reviewer", effective: "custom-reviewer", force: true, wantAction: gh.SyncChangeAdd},
		{name: "new weaker standard direct grant does not downgrade inherited access", source: "pull", effective: "admin", sameOrg: true, wantAction: gh.SyncChangeAdd},
		{name: "direct downgrade masked by parent", source: "pull", direct: "push", effective: "admin", sameOrg: true, wantSkip: true, wantWarning: "downgrade"},
		{name: "forced direct downgrade masked by parent", source: "pull", direct: "push", effective: "admin", sameOrg: true, force: true, wantAction: gh.SyncChangeUpdate},
		{name: "alias downgrade", source: "read", direct: "write", effective: "push", sameOrg: true, wantSkip: true, wantWarning: "downgrade"},
		{name: "forced alias downgrade", source: "read", direct: "write", effective: "push", sameOrg: true, force: true, wantAction: gh.SyncChangeUpdate},
		{name: "unknown existing direct permission", source: "pull", direct: "none", effective: "admin", sameOrg: true, wantSkip: true},
		{name: "forced unknown existing direct permission", source: "pull", direct: "none", effective: "admin", sameOrg: true, force: true, wantAction: gh.SyncChangeUpdate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, transport := newCopyFixtureClient(t, map[string][]copyFixtureResponse{
				"GET /repos/destination-org/sample": fixtureOK(`{"name":"sample"}`),
			})
			_, destination := fixtureTeams(g)
			var direct, effective []*github.Repository
			if tt.direct != "" {
				direct = []*github.Repository{testRepository("sample", tt.direct)}
			}
			if tt.effective != "" {
				effective = []*github.Repository{testRepository("sample", tt.effective)}
			}
			plan, err := planGrants(context.Background(), destination, []*github.Repository{testRepository("sample", tt.source)},
				direct, effective, nil, tt.sameOrg, tt.force)
			if err != nil {
				t.Fatal(err)
			}
			wantChange := tt.wantAction != ""
			if got := len(plan.Changes()) == 1; got != wantChange {
				t.Fatalf("changes = %#v, want action %q", plan.Changes(), tt.wantAction)
			}
			if wantChange && plan.Changes()[0].Action != tt.wantAction {
				t.Errorf("change = %#v, want action %q", plan.Changes()[0], tt.wantAction)
			}
			_, skipped := plan.skipped["sample"]
			if skipped != tt.wantSkip || (len(plan.Warnings()) == 1) != tt.wantSkip {
				t.Errorf("skipped = %t, warnings = %q; want skip %t", skipped, plan.Warnings(), tt.wantSkip)
			}
			warning := tt.wantWarning
			if warning == "" {
				warning = "incomparable"
			}
			if tt.wantSkip && !strings.Contains(plan.Warnings()[0], warning) {
				t.Errorf("warning %q does not contain %q", plan.Warnings(), warning)
			}
			if (len(transport.requests) == 1) != wantChange {
				t.Errorf("repository checks = %#v, want check %t", transport.requests, wantChange)
			}
		})
	}
}

func TestSafeToRemoveDirectPermission(t *testing.T) {
	for directRank, direct := range gh.PermissionsList {
		for effectiveRank, effective := range gh.PermissionsList {
			for parentRank, parent := range gh.PermissionsList {
				t.Run(direct+"/"+effective+"/"+parent, func(t *testing.T) {
					want := parentRank <= directRank && parentRank <= effectiveRank
					got := safeToRemoveDirectPermission(testRepository("sample", direct), testRepository("sample", effective), testRepository("sample", parent))
					if got != want {
						t.Errorf("safe = %t, want %t", got, want)
					}
				})
			}
		}

	}
	tests := []struct {
		name, direct, effective, parent string
		want                            bool
	}{
		{name: "missing parent", direct: "pull", effective: "pull"},
		{name: "missing effective", direct: "pull", parent: "admin"},
		{name: "unknown effective", direct: "pull", effective: "none", parent: "admin"},
		{name: "aliases", direct: "read", effective: "write", parent: "push", want: true},
		{name: "equal custom roles", direct: "custom-reviewer", effective: "custom-reviewer", parent: "custom-reviewer", want: true},
		{name: "custom direct role masked by standard parent", direct: "custom-reviewer", effective: "admin", parent: "admin"},
		{name: "custom parent masks standard direct role", direct: "pull", effective: "custom-reviewer", parent: "custom-reviewer"},
		{name: "different custom roles", direct: "custom-reviewer", effective: "custom-editor", parent: "custom-editor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var direct, effective, parent *github.Repository
			if tt.direct != "" {
				direct = testRepository("sample", tt.direct)
			}
			if tt.effective != "" {
				effective = testRepository("sample", tt.effective)
			}
			if tt.parent != "" {
				parent = testRepository("sample", tt.parent)
			}
			if got := safeToRemoveDirectPermission(direct, effective, parent); got != tt.want {
				t.Errorf("safe = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestPlanSyncProtectsChangesAcrossModes(t *testing.T) {
	modes := []struct {
		name string
		mode CopyMode
	}{
		{name: "direct", mode: CopyDirect},
		{name: "include inherited", mode: CopyWithInherited},
		{name: "effective", mode: CopyEffective},
	}
	for _, mode := range modes {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/force=%t", mode.name, force), func(t *testing.T) {
				responses := map[string][]copyFixtureResponse{
					"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
						{"name":"same","role_name":"push"},
						{"name":"upgrade","role_name":"admin"},
						{"name":"downgrade","role_name":"pull"},
						{"name":"custom-conflict","role_name":"custom-editor"},
						{"name":"custom-new","role_name":"custom-reviewer"},
						{"name":"equal-custom","role_name":"custom-reader"}
					]`),
					"GET /orgs/source-org/teams/source-team": fixtureOK(`{"slug":"source-team"}`),
					"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
						{"name":"same","role_name":"push"},
						{"name":"upgrade","role_name":"pull"},
						{"name":"downgrade","role_name":"push"},
						{"name":"custom-conflict","role_name":"pull"},
						{"name":"equal-custom","role_name":"custom-reader"},
						{"name":"extra","role_name":"admin"}
					]`),
					"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team"}`),
					"GET /repos/destination-org/upgrade":               fixtureOK(`{"name":"upgrade"}`),
					"GET /repos/destination-org/downgrade":             fixtureOK(`{"name":"downgrade"}`),
					"GET /repos/destination-org/custom-conflict":       fixtureOK(`{"name":"custom-conflict"}`),
					"GET /repos/destination-org/custom-new":            fixtureOK(`{"name":"custom-new"}`),
				}
				g, transport := newCopyFixtureClient(t, responses)
				source, destination := fixtureTeams(g)
				plan, err := PlanSync(context.Background(), source, destination, mode.mode, force)
				if err != nil {
					t.Fatal(err)
				}
				team := "destination-org/destination-team"
				want := []gh.SyncChange{{
					Destination: team, Action: gh.SyncChangeUpdate, Target: "upgrade", Permission: "admin", Current: "pull",
				}}
				if force {
					want = append(want,
						gh.SyncChange{Destination: team, Action: gh.SyncChangeUpdate, Target: "downgrade", Permission: "pull", Current: "push"},
						gh.SyncChange{Destination: team, Action: gh.SyncChangeUpdate, Target: "custom-conflict", Permission: "custom-editor", Current: "pull"},
					)
				}
				want = append(want, gh.SyncChange{
					Destination: team, Action: gh.SyncChangeAdd, Target: "custom-new", Permission: "custom-reviewer",
				})
				if force {
					want = append(want, gh.SyncChange{Destination: team, Action: gh.SyncChangeRemove, Target: "extra", Current: "admin"})
				}
				if got := plan.Changes(); !slices.Equal(got, want) {
					t.Fatalf("changes = %#v, want %#v", got, want)
				}
				for _, name := range []string{"downgrade", "custom-conflict", "equal-custom", "extra"} {
					_, skipped := plan.skipped[name]
					if skipped == force {
						t.Errorf("repository %s skipped = %t with force = %t", name, skipped, force)
					}
				}

				warnings := strings.Join(plan.Warnings(), "\n")
				if force && strings.Contains(warnings, "skipped") {
					t.Errorf("forced plan has a safety skip: %s", warnings)
				}
				if !force {
					for _, part := range []string{"downgrade would downgrade", "custom-conflict has incomparable", "equal-custom has incomparable", "extra cannot safely remove"} {
						if !strings.Contains(warnings, part) {
							t.Errorf("warnings %q do not contain %q", warnings, part)
						}
					}
					for _, request := range transport.requests {
						if request.path == "/repos/destination-org/downgrade" || request.path == "/repos/destination-org/custom-conflict" {
							t.Errorf("skipped permission was unnecessarily checked: %#v", request)
						}
					}
				}
				after := []*github.Repository{
					testRepository("same", "push"), testRepository("upgrade", "admin"),
					testRepository("downgrade", "push"), testRepository("custom-conflict", "pull"),
					testRepository("custom-new", "custom-reviewer"), testRepository("equal-custom", "custom-reader"),
					testRepository("extra", "admin"),
				}
				if force {
					after[2] = testRepository("downgrade", "pull")
					after[3] = testRepository("custom-conflict", "custom-editor")
					after = after[:len(after)-1]
				}
				verified, err := plan.verify(after, after)
				if err != nil {
					t.Fatal(err)
				}
				if !force {
					for _, warning := range verified {
						for _, name := range []string{"downgrade", "custom-conflict", "equal-custom", "extra"} {
							if strings.Contains(warning, name) {
								t.Errorf("skipped repository produced a verification warning: %s", warning)
							}
						}
					}
				}
			})
		}
	}
}

func TestSyncRedundantGrantRemovalProtectsEffectiveAccess(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprintf("force=%t", force), func(t *testing.T) {
			initial := `[{"id":2,"name":"sample","role_name":"admin","owner":{"login":"destination-org"}}]`
			after := initial
			initialTeams := `[{"slug":"destination-parent","permission":"push"},{"slug":"destination-team","permission":"admin"}]`
			afterTeams := initialTeams
			if force {
				after = `[{"id":2,"name":"sample","role_name":"push","owner":{"login":"destination-org"}}]`
				afterTeams = `[{"slug":"destination-parent","permission":"push"}]`
			}
			responses := map[string][]copyFixtureResponse{
				"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
					{"id":1,"name":"sample","role_name":"pull","owner":{"login":"source-org"}}
				]`),
				"GET /orgs/source-org/teams/source-team": fixtureOK(`{"slug":"source-team","parent":{"slug":"source-parent"}}`),
				"GET /orgs/source-org/teams/source-parent/repos": fixtureOK(`[
					{"id":1,"name":"sample","role_name":"pull","owner":{"login":"source-org"}}
				]`),
				"GET /repos/source-org/sample/teams": fixtureOK(`[{"slug":"source-parent","permission":"pull"}]`),
				"GET /orgs/destination-org/teams/destination-team/repos": {
					{status: http.StatusOK, body: initial},
					{status: http.StatusOK, body: initial},
					{status: http.StatusOK, body: after},
				},
				"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team","parent":{"slug":"destination-parent"}}`),
				"GET /orgs/destination-org/teams/destination-parent/repos": fixtureOK(`[
					{"id":2,"name":"sample","role_name":"push","owner":{"login":"destination-org"}}
				]`),
				"GET /repos/destination-org/sample/teams":                                          fixtureOK(afterTeams),
				"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/sample": {{status: http.StatusNoContent}},
			}
			g, transport := newCopyFixtureClient(t, responses)
			source, destination := fixtureTeams(g)
			plan, err := PlanSync(context.Background(), source, destination, CopyWithInherited, force)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(plan.Changes()) == 1; got != force {
				t.Fatalf("changes = %#v, want removal %t", plan.Changes(), force)
			}
			if force && plan.Changes()[0].Action != gh.SyncChangeRemove {
				t.Fatalf("changes = %#v, want removal", plan.Changes())
			}
			_, expectedAbsent := plan.expectedAbsent["sample"]
			if expectedAbsent != force {
				t.Errorf("expected absent = %t, want %t", expectedAbsent, force)
			}
			warnings, err := ApplySync(context.Background(), plan)
			if err != nil {
				t.Fatal(err)
			}
			if force && len(warnings) != 0 {
				t.Errorf("forced removal warnings = %q", warnings)
			}
			if !force && (len(warnings) != 1 || !strings.Contains(warnings[0], "cannot safely remove")) {
				t.Errorf("skipped removal warnings = %q", warnings)
			}
			var mutations []string
			for _, request := range transport.requests {
				if request.method != http.MethodGet {
					mutations = append(mutations, request.method+" "+request.path)
				}
			}
			if (len(mutations) == 1) != force {
				t.Errorf("mutations = %q, want removal %t", mutations, force)
			}
		})
	}
}

func TestSyncAllowsSameDestinationCustomRoleInheritedFromParent(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[]`),
		"GET /orgs/source-org/teams/source-team":       fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
			{"id":2,"name":"sample","role_name":"custom-reviewer","owner":{"login":"destination-org"}}
		]`),
		"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team","parent":{"slug":"destination-parent"}}`),
		"GET /orgs/destination-org/teams/destination-parent/repos": fixtureOK(`[
			{"id":2,"name":"sample","role_name":"custom-reviewer","owner":{"login":"destination-org"}}
		]`),
		"GET /repos/destination-org/sample/teams": {
			{status: http.StatusOK, body: `[{"slug":"destination-parent","permission":"custom-reviewer"},{"slug":"destination-team","permission":"custom-reviewer"}]`},
			{status: http.StatusOK, body: `[{"slug":"destination-parent","permission":"custom-reviewer"}]`},
		},
		"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/sample": {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanSync(context.Background(), source, destination, CopyDirect, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []gh.SyncChange{{
		Destination: "destination-org/destination-team", Action: gh.SyncChangeRemove,
		Target: "sample", Current: "custom-reviewer",
	}}
	if got := plan.Changes(); !slices.Equal(got, want) {
		t.Fatalf("changes = %#v, want %#v", got, want)
	}
	if got := plan.Warnings(); len(got) != 1 || !strings.Contains(got[0], "remains accessible through an inherited permission") {
		t.Errorf("planned warnings = %q", got)
	}
	warnings, err := ApplySync(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "remains accessible after synchronization") {
		t.Errorf("verification warnings = %q", warnings)
	}
	for _, request := range transport.requests {
		if request.method == http.MethodDelete {
			return
		}
	}
	t.Error("safe direct permission removal was not applied")
}

func TestPlanSyncStopsWhenDestinationParentDiscoveryFails(t *testing.T) {
	for _, mode := range []CopyMode{CopyDirect, CopyWithInherited, CopyEffective} {
		for _, force := range []bool{false, true} {
			t.Run(fmt.Sprintf("mode=%d/force=%t", mode, force), func(t *testing.T) {
				responses := map[string][]copyFixtureResponse{
					"GET /orgs/source-org/teams/source-team/repos":           fixtureOK(`[{"name":"sample","role_name":"pull"}]`),
					"GET /orgs/source-org/teams/source-team":                 fixtureOK(`{"slug":"source-team"}`),
					"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[{"name":"extra","role_name":"push"}]`),
					"GET /orgs/destination-org/teams/destination-team": {
						{status: http.StatusOK, body: `{"slug":"destination-team"}`},
						{status: http.StatusForbidden, body: `{"message":"Forbidden"}`},
					},
				}
				g, transport := newCopyFixtureClient(t, responses)
				source, destination := fixtureTeams(g)
				plan, err := PlanSync(context.Background(), source, destination, mode, force)
				if plan != nil || err == nil || !strings.Contains(err.Error(), "failed to get destination team") {
					t.Fatalf("plan = %#v, error = %v; want parent discovery failure", plan, err)
				}
				for _, request := range transport.requests {
					if request.method != http.MethodGet || request.path == "/repos/destination-org/sample" {
						t.Errorf("discovery failure led to an unexpected request: %#v", request)
					}
				}
			})
		}
	}
}
