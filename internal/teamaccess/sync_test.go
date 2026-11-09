package teamaccess

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

func TestSelectInheritedSync(t *testing.T) {
	source := []*github.Repository{
		testRepository("direct", "admin"), testRepository("no-parent", "pull"),
		testRepository("weaker-parent", "push"), testRepository("equal-parent", "pull"),
		testRepository("stronger-parent", "pull"), testRepository("custom-direct", "pull"),
		testRepository("custom-parent", "pull"),
	}
	direct := map[string]*github.Repository{"direct": testRepository("direct", "pull")}
	destination := map[string]*github.Repository{
		"no-parent":       testRepository("no-parent", "push"),
		"equal-parent":    testRepository("equal-parent", "pull"),
		"stronger-parent": testRepository("stronger-parent", "push"),
		"custom-direct":   testRepository("custom-direct", "custom-reader"),
	}
	parent := map[string]*github.Repository{
		"weaker-parent":   testRepository("weaker-parent", "pull"),
		"equal-parent":    testRepository("equal-parent", "pull"),
		"stronger-parent": testRepository("stronger-parent", "admin"),
		"custom-parent":   testRepository("custom-parent", "custom-reader"),
	}
	skipped := make(map[string]struct{})
	redundant := make(map[string]string)
	selected, warnings, err := selectInheritedSync(
		[]*github.Repository{direct["direct"]}, nil, source, direct, destination, parent, true, skipped, redundant,
	)
	if err != nil {
		t.Fatal(err)
	}
	var grants []string
	for _, repo := range selected {
		grants = append(grants, repo.GetName()+":"+gh.GetRepositoryPermissions(repo))
	}
	if want := []string{"direct:pull", "no-parent:pull", "weaker-parent:push"}; !slices.Equal(grants, want) {
		t.Errorf("selected = %q, want %q", grants, want)
	}
	if len(redundant) != 2 || redundant["equal-parent"] != "equal-parent" || redundant["stronger-parent"] != "stronger-parent" {
		t.Errorf("redundant = %#v", redundant)
	}
	if len(skipped) != 2 || len(warnings) != 2 || !strings.Contains(warnings[0], "custom-direct") || !strings.Contains(warnings[1], "custom-parent") {
		t.Errorf("skipped = %#v, warnings = %q", skipped, warnings)
	}
}

func TestSyncVerificationChecksDirectAndEffectivePermissions(t *testing.T) {
	plan := &SyncPlan{
		mode: CopyWithInherited, sameOrg: true,
		expected: map[string]*github.Repository{
			"source": testRepository("source", "admin"), "inherited": testRepository("inherited", "pull"),
			"skipped": testRepository("skipped", "custom-reader"),
		},
		sourceDirect:   map[string]*github.Repository{"source": testRepository("source", "pull")},
		expectedDirect: map[string]string{"source": "pull"},
		expectedAbsent: map[string]string{"inherited": "inherited"},
		skipped:        map[string]struct{}{"skipped": {}},
	}
	direct := []*github.Repository{testRepository("inherited", "pull")}
	effective := []*github.Repository{
		testRepository("source", "admin"), testRepository("inherited", "admin"),
		testRepository("skipped", "custom-writer"), testRepository("extra", "push"),
	}
	warnings, err := plan.verify(direct, effective)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"source has direct permission none", "inherited still has direct access",
		"source has effective permission admin", "extra remains accessible",
	} {
		if !slices.ContainsFunc(warnings, func(w string) bool { return strings.Contains(w, want) }) {
			t.Errorf("warnings %q do not contain %q", warnings, want)
		}
	}
	for _, warning := range warnings {
		if strings.Contains(warning, "skipped") || strings.Contains(warning, "inherited has effective") {
			t.Errorf("unexpected warning: %s", warning)
		}
	}
}

func TestPlanSyncRejectsInconsistentSourceSnapshot(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": {
			{status: http.StatusOK, body: `[{"name":"sample","role_name":"pull"}]`},
			{status: http.StatusOK, body: `[]`},
		},
		"GET /orgs/source-org/teams/source-team":                 fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[]`),
		"GET /orgs/destination-org/teams/destination-team":       fixtureOK(`{"slug":"destination-team"}`),
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanSync(context.Background(), source, destination, CopyWithInherited)
	if plan != nil || err == nil || !strings.Contains(err.Error(), "missing from effective permissions") {
		t.Fatalf("PlanSync() = %#v, %v; want inconsistent source error", plan, err)
	}
	for _, request := range transport.requests {
		if request.method != http.MethodGet {
			t.Fatalf("unexpected mutation: %#v", request)
		}
	}
}

func TestPlanSyncPartialValidationCannotRevoke(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
			{"name":"alpha","role_name":"pull"},{"name":"failure","role_name":"push"}
		]`),
		"GET /orgs/source-org/teams/source-team": fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
			{"name":"extra","role_name":"admin"}
		]`),
		"GET /orgs/destination-org/teams/destination-team":                             fixtureOK(`{"slug":"destination-team"}`),
		"GET /repos/destination-org/alpha":                                             fixtureOK(`{"name":"alpha"}`),
		"GET /repos/destination-org/failure":                                           {{status: http.StatusForbidden, body: `{"message":"Forbidden"}`}},
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/alpha": {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanSync(context.Background(), source, destination, CopyWithInherited)
	if err == nil || plan == nil || plan.complete {
		t.Fatalf("PlanSync() = %#v, %v; want partial plan", plan, err)
	}
	if len(plan.Changes()) != 1 || plan.Changes()[0].Target != "alpha" {
		t.Fatalf("partial changes = %#v", plan.Changes())
	}
	if _, err := ApplySync(context.Background(), plan); err == nil {
		t.Fatal("incomplete sync plan was accepted")
	}
	if err := ApplySyncGrants(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	for _, request := range transport.requests {
		if request.method == http.MethodDelete {
			t.Fatalf("partial plan revoked access: %#v", request)
		}
	}
}

func TestSyncInheritedParentRemovesRedundantDirectGrant(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
			{"id":1,"name":"sample","role_name":"pull","owner":{"login":"source-org"}}
		]`),
		"GET /orgs/source-org/teams/source-team": fixtureOK(`{"slug":"source-team","parent":{"slug":"source-parent"}}`),
		"GET /orgs/source-org/teams/source-parent/repos": fixtureOK(`[
			{"id":1,"name":"sample","role_name":"pull","owner":{"login":"source-org"}}
		]`),
		"GET /repos/source-org/sample/teams": fixtureOK(`[{"slug":"source-parent","permission":"pull"}]`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
			{"id":2,"name":"sample","role_name":"admin","owner":{"login":"destination-org"}}
		]`),
		"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team","parent":{"slug":"destination-parent"}}`),
		"GET /orgs/destination-org/teams/destination-parent/repos": fixtureOK(`[
			{"id":2,"name":"sample","role_name":"admin","owner":{"login":"destination-org"}}
		]`),
		"GET /repos/destination-org/sample/teams": {
			{status: http.StatusOK, body: `[{"slug":"destination-parent","permission":"admin"},{"slug":"destination-team","permission":"pull"}]`},
			{status: http.StatusOK, body: `[{"slug":"destination-parent","permission":"admin"}]`},
		},
		"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/sample": {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanSync(context.Background(), source, destination, CopyWithInherited)
	if err != nil {
		t.Fatal(err)
	}
	want := []gh.SyncChange{{
		Destination: "destination-org/destination-team", Action: gh.SyncChangeRemove,
		Target: "sample", Current: "pull",
	}}
	if got := plan.Changes(); !slices.Equal(got, want) {
		t.Fatalf("changes = %#v, want %#v", got, want)
	}
	warnings, err := ApplySync(context.Background(), plan)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("ApplySync() = %q, %v; want no warnings", warnings, err)
	}
	var mutations []string
	for _, request := range transport.requests {
		if request.method != http.MethodGet {
			mutations = append(mutations, request.method+" "+request.path)
		}
	}
	if want := []string{"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/sample"}; !slices.Equal(mutations, want) {
		t.Errorf("mutations = %q, want %q", mutations, want)
	}
}

func TestSyncEmptySourceRemovesDirectGrantsButWarnsOnInheritedAccess(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[]`),
		"GET /orgs/source-org/teams/source-team":       fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
			{"id":2,"name":"extra","role_name":"admin","owner":{"login":"destination-org"}},
			{"id":3,"name":"parent-only","role_name":"pull","owner":{"login":"destination-org"}}
		]`),
		"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team","parent":{"slug":"destination-parent"}}`),
		"GET /orgs/destination-org/teams/destination-parent/repos": fixtureOK(`[
			{"id":2,"name":"extra","role_name":"admin","owner":{"login":"destination-org"}},
			{"id":3,"name":"parent-only","role_name":"pull","owner":{"login":"destination-org"}}
		]`),
		"GET /repos/destination-org/extra/teams": {
			{status: http.StatusOK, body: `[{"slug":"destination-parent","permission":"admin"},{"slug":"destination-team","permission":"pull"}]`},
			{status: http.StatusOK, body: `[{"slug":"destination-parent","permission":"admin"}]`},
		},
		"GET /repos/destination-org/parent-only/teams":                                    fixtureOK(`[{"slug":"destination-parent","permission":"pull"}]`),
		"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/extra": {{status: http.StatusNoContent}},
	}
	g, _ := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanSync(context.Background(), source, destination, CopyWithInherited)
	if err != nil {
		t.Fatal(err)
	}
	want := []gh.SyncChange{{
		Destination: "destination-org/destination-team", Action: gh.SyncChangeRemove,
		Target: "extra", Current: "pull",
	}}
	if got := plan.Changes(); !slices.Equal(got, want) {
		t.Fatalf("changes = %#v, want %#v", got, want)
	}
	warnings, err := ApplySync(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"extra", "parent-only"} {
		if !slices.ContainsFunc(warnings, func(w string) bool { return strings.Contains(w, "repository "+name+" remains accessible") }) {
			t.Errorf("warnings %q do not report inherited access to %s", warnings, name)
		}
	}
}

func TestSyncMissingRepositoryDoesNotRevokeExistingDirectGrant(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos":           fixtureOK(`[{"name":"sample","role_name":"pull"}]`),
		"GET /orgs/source-org/teams/source-team":                 fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[{"name":"sample","role_name":"admin"}]`),
		"GET /orgs/destination-org/teams/destination-team":       fixtureOK(`{"slug":"destination-team"}`),
		"GET /repos/destination-org/sample":                      {{status: http.StatusNotFound, body: `{"message":"Not Found"}`}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanSync(context.Background(), source, destination, CopyWithInherited)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Changes()) != 0 || len(plan.Warnings()) != 1 {
		t.Fatalf("changes = %#v, warnings = %q", plan.Changes(), plan.Warnings())
	}
	warnings, err := ApplySync(context.Background(), plan)
	if err != nil || len(warnings) != 1 || !strings.Contains(warnings[0], "404") {
		t.Fatalf("ApplySync() = %q, %v; want only skipped warning", warnings, err)
	}
	for _, request := range transport.requests {
		if request.method != http.MethodGet {
			t.Fatalf("skipped repository was modified: %#v", request)
		}
	}
}

func TestApplySyncMutatesThenVerifies(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/alpha":    {{status: http.StatusNoContent}},
		"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/extra": {{status: http.StatusNoContent}},
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
			{"name":"alpha","role_name":"pull"}
		]`),
		"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team"}`),
	}
	g, transport := newCopyFixtureClient(t, responses)
	_, destination := fixtureTeams(g)
	plan := &SyncPlan{
		grants: &CopyPlan{destination: destination, changes: []gh.SyncChange{{
			Action: gh.SyncChangeAdd, Target: "alpha", Permission: "pull",
		}}},
		removals:       []gh.SyncChange{{Action: gh.SyncChangeRemove, Target: "extra"}},
		expected:       map[string]*github.Repository{"alpha": testRepository("alpha", "pull")},
		expectedDirect: map[string]string{"alpha": "pull"},
		expectedAbsent: map[string]string{"extra": "extra"},
		sourceDirect:   map[string]*github.Repository{"alpha": testRepository("alpha", "pull")},
		mode:           CopyEffective,
		sameOrg:        true,
		complete:       true,
	}
	warnings, err := ApplySync(context.Background(), plan)
	if err != nil || len(warnings) != 0 {
		t.Fatalf("ApplySync() = %q, %v", warnings, err)
	}
	var methods []string
	for _, request := range transport.requests {
		methods = append(methods, request.method)
	}
	if want := []string{"PUT", "DELETE", "GET", "GET", "GET"}; !slices.Equal(methods, want) {
		t.Errorf("request sequence = %q, want %q", methods, want)
	}
}

func TestApplySyncStopsOnMutationFailure(t *testing.T) {
	tests := []struct {
		name         string
		putStatus    int
		deleteStatus int
		wantMethods  []string
	}{
		{name: "grant failure", putStatus: http.StatusForbidden, deleteStatus: http.StatusNoContent, wantMethods: []string{"PUT"}},
		{name: "revoke failure", putStatus: http.StatusNoContent, deleteStatus: http.StatusForbidden, wantMethods: []string{"PUT", "DELETE"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responses := map[string][]copyFixtureResponse{
				"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/alpha":    {{status: tt.putStatus, body: `{"message":"failure"}`}},
				"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/extra": {{status: tt.deleteStatus, body: `{"message":"failure"}`}},
				"DELETE /orgs/destination-org/teams/destination-team/repos/destination-org/later": {{status: http.StatusNoContent}},
			}
			g, transport := newCopyFixtureClient(t, responses)
			_, destination := fixtureTeams(g)
			plan := &SyncPlan{
				grants:   &CopyPlan{destination: destination, changes: []gh.SyncChange{{Target: "alpha", Permission: "pull"}}},
				removals: []gh.SyncChange{{Target: "extra"}, {Target: "later"}},
				complete: true,
			}
			if _, err := ApplySync(context.Background(), plan); err == nil {
				t.Fatal("expected mutation failure")
			}
			var methods []string
			for _, request := range transport.requests {
				methods = append(methods, request.method)
			}
			if !slices.Equal(methods, tt.wantMethods) {
				t.Errorf("methods = %q, want %q", methods, tt.wantMethods)
			}
		})
	}
}

func TestApplySyncVerifiesNoOpPlan(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/destination-org/teams/destination-team/repos": {{status: http.StatusBadGateway, body: `{"message":"Unavailable"}`}},
	}
	g, _ := newCopyFixtureClient(t, responses)
	_, destination := fixtureTeams(g)
	plan := &SyncPlan{grants: &CopyPlan{destination: destination}, complete: true}
	_, err := ApplySync(context.Background(), plan)
	if err == nil || !strings.Contains(err.Error(), "changes may already have been applied") {
		t.Fatalf("ApplySync() error = %v, want explicit verification failure", err)
	}
}
