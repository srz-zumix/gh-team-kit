package teamaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/gh/client"
)

type copyFixtureResponse struct {
	status int
	body   string
}

type copyFixtureRequest struct {
	method string
	path   string
	body   string
}

type copyFixtureTransport struct {
	mu        sync.Mutex
	responses map[string][]copyFixtureResponse
	calls     map[string]int
	requests  []copyFixtureRequest
}

func (tr *copyFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
	}
	key := req.Method + " " + req.URL.Path
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.requests = append(tr.requests, copyFixtureRequest{req.Method, req.URL.Path, string(body)})
	responses := tr.responses[key]
	if len(responses) == 0 {
		return nil, fmt.Errorf("unexpected API request %s", key)
	}
	index := tr.calls[key]
	tr.calls[key]++
	if index >= len(responses) {
		index = len(responses) - 1
	}
	response := responses[index]
	return &http.Response{
		StatusCode: response.status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(response.body)),
		Request:    req,
	}, nil
}

func newCopyFixtureClient(t *testing.T, responses map[string][]copyFixtureResponse) (*gh.GitHubClient, *copyFixtureTransport) {
	t.Helper()
	transport := &copyFixtureTransport{responses: responses, calls: make(map[string]int)}
	base := "https://api.example.test/"
	api, err := github.NewClient(github.WithHTTPClient(&http.Client{Transport: transport}), github.WithURLs(&base, nil))
	if err != nil {
		t.Fatal(err)
	}
	g, err := client.NewClient(api)
	if err != nil {
		t.Fatal(err)
	}
	return g, transport
}

func fixtureOK(body string) []copyFixtureResponse {
	return []copyFixtureResponse{{status: http.StatusOK, body: body}}
}

func fixtureTeams(g *gh.GitHubClient) (Team, Team) {
	return Team{Client: g, Repository: repository.Repository{Owner: "source-org"}, Slug: "source-team"},
		Team{Client: g, Repository: repository.Repository{Owner: "destination-org"}, Slug: "destination-team"}
}

func TestPlanCopyValidatesOnlyFinalGrants(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
			{"id":1,"name":"equivalent","role_name":"admin","owner":{"login":"source-org"}}
		]`),
		"GET /orgs/source-org/teams/source-team": fixtureOK(`{"slug":"source-team","parent":{"slug":"source-parent"}}`),
		"GET /orgs/source-org/teams/source-parent/repos": fixtureOK(`[
			{"id":1,"name":"equivalent","role_name":"admin","owner":{"login":"source-org"}}
		]`),
		"GET /repos/source-org/equivalent/teams": fixtureOK(`[
			{"slug":"source-parent","permission":"admin"},{"slug":"source-team","permission":"pull"}
		]`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
			{"name":"equivalent","role_name":"admin","owner":{"login":"destination-org"}}
		]`),
		"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team"}`),
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanCopy(context.Background(), source, destination, CopyEffective, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.Changes(); len(got) != 0 {
		t.Fatalf("changes = %#v, want none", got)
	}
	for _, request := range transport.requests {
		if request.path == "/repos/destination-org/equivalent" {
			t.Fatalf("unchanged effective permission was unnecessarily validated: %#v", request)
		}
	}
}

func TestPlanCopyMissingRepositoriesAndPartialErrors(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
			{"name":"alpha","role_name":"push"},
			{"name":"missing","role_name":"pull"},
			{"name":"failure","role_name":"admin"},
			{"name":"later","role_name":"pull"}
		]`),
		"GET /orgs/source-org/teams/source-team":                                       fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos":                       fixtureOK(`[]`),
		"GET /orgs/destination-org/teams/destination-team":                             fixtureOK(`{"slug":"destination-team"}`),
		"GET /repos/destination-org/alpha":                                             fixtureOK(`{"name":"alpha"}`),
		"GET /repos/destination-org/missing":                                           {{status: http.StatusNotFound, body: `{"message":"Not Found"}`}},
		"GET /repos/destination-org/failure":                                           {{status: http.StatusForbidden, body: `{"message":"Forbidden"}`}},
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/alpha": {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	destination.Repository.Host = "github.example.test"
	plan, err := PlanCopy(context.Background(), source, destination, CopyDirect, false)
	if err == nil || !strings.Contains(err.Error(), "failed to check destination repository destination-org/failure") {
		t.Fatalf("PlanCopy() error = %v, want destination lookup failure", err)
	}
	if plan == nil {
		t.Fatal("PlanCopy() returned no partial plan")
	}
	want := []gh.SyncChange{{
		Destination: "github.example.test/destination-org/destination-team",
		Action:      gh.SyncChangeAdd,
		Target:      "alpha",
		Permission:  "push",
	}}
	if got := plan.Changes(); !slices.Equal(got, want) {
		t.Errorf("changes = %#v, want %#v", got, want)
	}
	if got := plan.Warnings(); len(got) != 1 || !strings.Contains(got[0], "missing") || !strings.Contains(got[0], "404") {
		t.Errorf("warnings = %q, want missing repository warning", got)
	}
	for _, request := range transport.requests {
		if request.method == http.MethodPut || request.path == "/repos/destination-org/later" {
			t.Fatalf("planning made an unexpected request: %#v", request)
		}
	}

	if err := ApplyCopy(context.Background(), plan); err != nil {
		t.Fatalf("ApplyCopy() error = %v", err)
	}
	var putRequests []copyFixtureRequest
	for _, request := range transport.requests {
		if request.method == http.MethodPut {
			putRequests = append(putRequests, request)
		}
	}
	if len(putRequests) != 1 || putRequests[0].path != "/orgs/destination-org/teams/destination-team/repos/destination-org/alpha" {
		t.Fatalf("PUT requests = %#v, want only alpha", putRequests)
	}
	var body struct {
		Permission string `json:"permission"`
	}
	if err := json.Unmarshal([]byte(putRequests[0].body), &body); err != nil {
		t.Fatal(err)
	}
	if body.Permission != "push" {
		t.Errorf("applied permission = %q, want push", body.Permission)
	}
}

func TestPlanCopyContinuesAfterMissingRepository(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
			{"name":"alpha","role_name":"pull"},
			{"name":"missing","role_name":"push"},
			{"name":"beta","role_name":"admin"}
		]`),
		"GET /orgs/source-org/teams/source-team":                                       fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos":                       fixtureOK(`[]`),
		"GET /orgs/destination-org/teams/destination-team":                             fixtureOK(`{"slug":"destination-team"}`),
		"GET /repos/destination-org/alpha":                                             fixtureOK(`{"name":"alpha"}`),
		"GET /repos/destination-org/missing":                                           {{status: http.StatusNotFound, body: `{"message":"Not Found"}`}},
		"GET /repos/destination-org/beta":                                              fixtureOK(`{"name":"beta"}`),
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/alpha": {{status: http.StatusNoContent}},
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/beta":  {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanCopy(context.Background(), source, destination, CopyDirect, false)
	if err != nil {
		t.Fatal(err)
	}
	var targets []string
	for _, change := range plan.Changes() {
		targets = append(targets, change.Target+":"+change.Permission)
	}
	if !slices.Equal(targets, []string{"alpha:pull", "beta:admin"}) {
		t.Errorf("changes = %q", targets)
	}
	if len(plan.Warnings()) != 1 {
		t.Errorf("warnings = %q, want one", plan.Warnings())
	}
	if err := ApplyCopy(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	for _, request := range transport.requests {
		if request.method == http.MethodPut && strings.HasSuffix(request.path, "/missing") {
			t.Fatalf("missing repository was granted access: %#v", request)
		}
	}
}

func TestPlanCopyRejectsEmptyDestinationRepositoryResponse(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos":           fixtureOK(`[{"name":"alpha","role_name":"pull"}]`),
		"GET /orgs/source-org/teams/source-team":                 fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[]`),
		"GET /orgs/destination-org/teams/destination-team":       fixtureOK(`{"slug":"destination-team"}`),
		"GET /repos/destination-org/alpha":                       fixtureOK(`null`),
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanCopy(context.Background(), source, destination, CopyDirect, false)
	if err == nil || !strings.Contains(err.Error(), "destination repository destination-org/alpha was not returned") {
		t.Fatalf("PlanCopy() error = %v, want an empty response error", err)
	}
	if plan == nil || len(plan.Changes()) != 0 {
		t.Fatalf("plan = %#v, want no validated grants", plan)
	}
	for _, request := range transport.requests {
		if request.method == http.MethodPut {
			t.Fatalf("planning applied a grant: %#v", request)
		}
	}
}

func TestApplyCopyStopsAfterGrantError(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
			{"name":"alpha","role_name":"pull"},{"name":"beta","role_name":"push"}
		]`),
		"GET /orgs/source-org/teams/source-team":                                       fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos":                       fixtureOK(`[]`),
		"GET /orgs/destination-org/teams/destination-team":                             fixtureOK(`{"slug":"destination-team"}`),
		"GET /repos/destination-org/alpha":                                             fixtureOK(`{"name":"alpha"}`),
		"GET /repos/destination-org/beta":                                              fixtureOK(`{"name":"beta"}`),
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/alpha": {{status: http.StatusForbidden, body: `{"message":"Forbidden"}`}},
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/beta":  {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanCopy(context.Background(), source, destination, CopyDirect, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyCopy(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "failed to grant team destination-team access") {
		t.Fatalf("ApplyCopy() error = %v, want grant failure", err)
	}
	for _, request := range transport.requests {
		if request.method == http.MethodPut && strings.HasSuffix(request.path, "/beta") {
			t.Fatalf("grant continued after a failure: %#v", request)
		}
	}
}

func TestPlanCopyParentDiscoveryFailureDoesNotReturnPartialPlan(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos":           fixtureOK(`[{"name":"alpha","role_name":"pull"}]`),
		"GET /orgs/source-org/teams/source-team":                 fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[]`),
		"GET /orgs/destination-org/teams/destination-team": {
			{status: http.StatusOK, body: `{"slug":"destination-team"}`},
			{status: http.StatusBadGateway, body: `{"message":"Unavailable"}`},
		},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanCopy(context.Background(), source, destination, CopyEffective, false)
	if err == nil || !strings.Contains(err.Error(), "failed to get destination team") || plan != nil {
		t.Fatalf("PlanCopy() = %#v, %v; want nil plan and parent discovery error", plan, err)
	}
	for _, request := range transport.requests {
		if request.method == http.MethodPut {
			t.Fatalf("discovery error caused a write: %#v", request)
		}
	}
}

func TestPlanCopyUpdatesDestinationDirectRoleMaskedByParent(t *testing.T) {
	responses := map[string][]copyFixtureResponse{
		"GET /orgs/source-org/teams/source-team/repos": fixtureOK(`[
				{"id":1,"name":"masked","role_name":"admin","owner":{"login":"source-org"}}
			]`),
		"GET /orgs/source-org/teams/source-team": fixtureOK(`{"slug":"source-team"}`),
		"GET /orgs/destination-org/teams/destination-team/repos": fixtureOK(`[
				{"id":2,"name":"masked","role_name":"admin","owner":{"login":"destination-org"}}
			]`),
		"GET /orgs/destination-org/teams/destination-team": fixtureOK(`{"slug":"destination-team","parent":{"slug":"destination-parent"}}`),
		"GET /orgs/destination-org/teams/destination-parent/repos": fixtureOK(`[
				{"id":2,"name":"masked","role_name":"admin","owner":{"login":"destination-org"}}
			]`),
		"GET /repos/destination-org/masked/teams": fixtureOK(`[
				{"slug":"destination-parent","permission":"admin"},{"slug":"destination-team","permission":"pull"}
			]`),
		"GET /repos/destination-org/masked":                                             fixtureOK(`{"name":"masked"}`),
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/masked": {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	plan, err := PlanCopy(context.Background(), source, destination, CopyDirect, false)
	if err != nil {
		t.Fatal(err)
	}
	want := []gh.SyncChange{{
		Destination: "destination-org/destination-team",
		Action:      gh.SyncChangeUpdate,
		Target:      "masked",
		Permission:  "admin",
		Current:     "pull",
	}}
	if got := plan.Changes(); !slices.Equal(got, want) {
		t.Fatalf("changes = %#v, want %#v", got, want)
	}
	if err := ApplyCopy(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	last := transport.requests[len(transport.requests)-1]
	var body struct {
		Permission string `json:"permission"`
	}
	if last.method != http.MethodPut || json.Unmarshal([]byte(last.body), &body) != nil || body.Permission != "admin" {
		t.Fatalf("last request = %#v, want a PUT granting admin", last)
	}
}

func TestInheritedSourceAgainstStrongerDestinationDirectRole(t *testing.T) {
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
		"GET /orgs/destination-org/teams/destination-team":                              fixtureOK(`{"slug":"destination-team"}`),
		"GET /repos/destination-org/sample":                                             fixtureOK(`{"name":"sample"}`),
		"PUT /orgs/destination-org/teams/destination-team/repos/destination-org/sample": {{status: http.StatusNoContent}},
	}
	g, transport := newCopyFixtureClient(t, responses)
	source, destination := fixtureTeams(g)
	includePlan, err := PlanCopy(context.Background(), source, destination, CopyWithInherited, false)
	if err != nil || len(includePlan.Changes()) != 0 || len(includePlan.Warnings()) != 0 {
		t.Fatalf("include inherited plan = %#v, %v; want no changes or warnings", includePlan, err)
	}
	for _, request := range transport.requests {
		if request.path == "/repos/destination-org/sample" {
			t.Fatalf("unchanged inherited permission was unnecessarily validated: %#v", request)
		}
	}
	protectedPlan, err := PlanCopy(context.Background(), source, destination, CopyEffective, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(protectedPlan.Changes()) != 0 || len(protectedPlan.Warnings()) != 1 ||
		!strings.Contains(protectedPlan.Warnings()[0], "would downgrade direct permission from admin to pull") {
		t.Fatalf("protected effective plan changes = %#v, warnings = %q", protectedPlan.Changes(), protectedPlan.Warnings())
	}
	for _, request := range transport.requests {
		if request.path == "/repos/destination-org/sample" {
			t.Fatalf("skipped downgrade was unnecessarily validated: %#v", request)
		}
	}
	effectivePlan, err := PlanCopy(context.Background(), source, destination, CopyEffective, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []gh.SyncChange{{
		Destination: "destination-org/destination-team", Action: gh.SyncChangeUpdate,
		Target: "sample", Permission: "pull", Current: "admin",
	}}
	if got := effectivePlan.Changes(); !slices.Equal(got, want) {
		t.Fatalf("effective changes = %#v, want %#v", got, want)
	}
	if got := effectivePlan.Warnings(); len(got) != 0 {
		t.Fatalf("effective warnings = %q, want none without a parent", got)
	}
	if err := ApplyCopy(context.Background(), effectivePlan); err != nil {
		t.Fatal(err)
	}
	last := transport.requests[len(transport.requests)-1]
	var body struct {
		Permission string `json:"permission"`
	}
	if last.method != http.MethodPut || json.Unmarshal([]byte(last.body), &body) != nil || body.Permission != "pull" {
		t.Fatalf("last request = %#v, want a PUT granting pull", last)
	}
}

func TestInheritedCustomRoleComparisonUsesResolvedOrganization(t *testing.T) {
	t.Setenv("GH_HOST", "github.example.test")
	tests := []struct {
		name        string
		destination string
		wantWarning bool
	}{
		{name: "same organization and default host", destination: "source-org"},
		{name: "different organization", destination: "destination-org", wantWarning: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceRepo := `[{"id":1,"name":"custom","role_name":"custom-reader","owner":{"login":"source-org"}}]`
			destinationRepo := fmt.Sprintf(`[{"id":1,"name":"custom","role_name":"custom-reader","owner":{"login":%q}}]`, tt.destination)
			responses := map[string][]copyFixtureResponse{
				"GET /orgs/source-org/teams/source-team/repos":                    fixtureOK(sourceRepo),
				"GET /orgs/source-org/teams/source-parent/repos":                  fixtureOK(sourceRepo),
				"GET /orgs/source-org/teams/source-team":                          fixtureOK(`{"slug":"source-team","parent":{"slug":"source-parent"}}`),
				"GET /orgs/" + tt.destination + "/teams/destination-team/repos":   fixtureOK(destinationRepo),
				"GET /orgs/" + tt.destination + "/teams/destination-parent/repos": fixtureOK(destinationRepo),
				"GET /orgs/" + tt.destination + "/teams/destination-team":         fixtureOK(`{"slug":"destination-team","parent":{"slug":"destination-parent"}}`),
				"GET /repos/source-org/custom/teams": fixtureOK(`[
						{"slug":"source-parent","permission":"custom-reader"},{"slug":"destination-parent","permission":"custom-reader"}
					]`),
				"GET /repos/" + tt.destination + "/custom/teams": fixtureOK(`[
						{"slug":"source-parent","permission":"custom-reader"},{"slug":"destination-parent","permission":"custom-reader"}
					]`),
			}
			g, _ := newCopyFixtureClient(t, responses)
			source, destination := fixtureTeams(g)
			destination.Repository = repository.Repository{Host: "github.example.test", Owner: tt.destination}
			plan, err := PlanCopy(context.Background(), source, destination, CopyWithInherited, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Changes()) != 0 {
				t.Fatalf("changes = %#v, want none", plan.Changes())
			}
			if gotWarning := len(plan.Warnings()) > 0; gotWarning != tt.wantWarning {
				t.Errorf("warnings = %q, want warning %t", plan.Warnings(), tt.wantWarning)
			}
		})
	}
}
