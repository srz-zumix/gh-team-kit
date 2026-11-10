package teamaccess

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

func testRepository(name, permission string) *github.Repository {
	return &github.Repository{Name: github.Ptr(name), RoleName: github.Ptr(permission)}
}

func TestCopyRepositorySelection(t *testing.T) {
	tests := []struct {
		name                 string
		mode                 CopyMode
		sameOrg              bool
		force                bool
		sourceDirect         []*github.Repository
		sourceEffective      []*github.Repository
		destinationDirect    []*github.Repository
		destinationEffective []*github.Repository
		destinationParent    []*github.Repository
		wantGrants           []string
		wantWarnings         []string
	}{
		{
			name:    "direct selection retains conflicts and equal inherited access",
			mode:    CopyDirect,
			sameOrg: true,
			sourceDirect: []*github.Repository{
				testRepository("inherited-equal", "pull"),
				testRepository("downgrade", "pull"),
				testRepository("upgrade", "admin"),
				testRepository("same-direct", "push"),
				testRepository("alias", "write"),
			},
			destinationDirect: []*github.Repository{
				testRepository("downgrade", "admin"),
				testRepository("upgrade", "pull"),
				testRepository("same-direct", "push"),
				testRepository("alias", "pull"),
			},
			destinationEffective: []*github.Repository{
				testRepository("inherited-equal", "pull"),
				testRepository("downgrade", "admin"),
				testRepository("upgrade", "pull"),
				testRepository("same-direct", "push"),
				testRepository("alias", "pull"),
			},
			wantGrants: []string{"inherited-equal:pull", "downgrade:pull", "upgrade:admin", "alias:push"},
		},
		{
			name:         "include inherited uses direct role and supplements only missing access",
			mode:         CopyWithInherited,
			sameOrg:      true,
			sourceDirect: []*github.Repository{testRepository("both", "pull")},
			sourceEffective: []*github.Repository{
				testRepository("both", "admin"),
				testRepository("weaker", "push"),
				testRepository("equal", "read"),
				testRepository("stronger", "pull"),
				testRepository("missing", "pull"),
			},
			destinationDirect: []*github.Repository{testRepository("weaker", "pull")},
			destinationEffective: []*github.Repository{
				testRepository("weaker", "pull"),
				testRepository("equal", "pull"),
				testRepository("stronger", "admin"),
			},
			wantGrants: []string{"both:pull", "weaker:push", "missing:pull"},
		},
		{
			name:    "effective role materializes direct sources and warns on stronger parents",
			mode:    CopyEffective,
			sameOrg: true,
			sourceDirect: []*github.Repository{
				testRepository("masked", "pull"),
				testRepository("materialize", "pull"),
				testRepository("overwrite", "pull"),
			},
			sourceEffective: []*github.Repository{
				testRepository("masked", "admin"),
				testRepository("materialize", "pull"),
				testRepository("overwrite", "pull"),
				testRepository("inherited-equal", "pull"),
				testRepository("strong-parent", "pull"),
				testRepository("already-direct", "pull"),
			},
			destinationDirect: []*github.Repository{
				testRepository("overwrite", "push"),
				testRepository("already-direct", "pull"),
			},
			destinationEffective: []*github.Repository{
				testRepository("materialize", "pull"),
				testRepository("overwrite", "pull"),
				testRepository("inherited-equal", "pull"),
				testRepository("strong-parent", "admin"),
				testRepository("already-direct", "admin"),
			},
			destinationParent: []*github.Repository{
				testRepository("materialize", "pull"),
				testRepository("overwrite", "pull"),
				testRepository("inherited-equal", "pull"),
				testRepository("strong-parent", "admin"),
				testRepository("already-direct", "admin"),
			},
			wantGrants: []string{"masked:admin", "materialize:pull", "overwrite:pull", "strong-parent:pull"},
			wantWarnings: []string{
				"repository strong-parent inherits destination parent permission admin, which is stronger than pull; effective access will remain stronger",
				"repository already-direct inherits destination parent permission admin, which is stronger than pull; effective access will remain stronger",
			},
		},
		{
			name:    "custom inherited roles compare only within the same organization",
			mode:    CopyWithInherited,
			sameOrg: true,
			sourceEffective: []*github.Repository{
				testRepository("equal", "custom-reader"),
				testRepository("different", "custom-reader"),
				testRepository("standard", "custom-reader"),
				testRepository("reverse", "pull"),
				testRepository("missing", "custom-reader"),
			},
			destinationEffective: []*github.Repository{
				testRepository("equal", "custom-reader"),
				testRepository("different", "custom-writer"),
				testRepository("standard", "admin"),
				testRepository("reverse", "custom-reader"),
			},
			wantGrants: []string{"missing:custom-reader"},
			wantWarnings: []string{
				"repository different has incomparable source permission custom-reader and destination permission custom-writer; skipped; use --force to override",
				"repository standard has incomparable source permission custom-reader and destination permission admin; skipped; use --force to override",
				"repository reverse has incomparable source permission pull and destination permission custom-reader; skipped; use --force to override",
			},
		},
		{
			name:                 "same custom role in another organization is not comparable",
			mode:                 CopyEffective,
			sourceEffective:      []*github.Repository{testRepository("custom", "custom-reader")},
			destinationEffective: []*github.Repository{testRepository("custom", "custom-reader")},
			wantWarnings:         []string{"repository custom has incomparable source permission custom-reader and destination permission custom-reader; skipped; use --force to override"},
		},
		{
			name:                 "custom source direct permission is selected before grant safety",
			mode:                 CopyWithInherited,
			sourceDirect:         []*github.Repository{testRepository("custom", "custom-reader")},
			sourceEffective:      []*github.Repository{testRepository("custom", "custom-reader")},
			destinationDirect:    []*github.Repository{testRepository("custom", "custom-writer")},
			destinationEffective: []*github.Repository{testRepository("custom", "custom-writer")},
			wantGrants:           []string{"custom:custom-reader"},
			wantWarnings:         []string{"repository custom uses custom role custom-reader; definitions may differ in the destination organization"},
		},
		{
			name:            "custom inherited role copied to another organization warns about its definition",
			mode:            CopyWithInherited,
			sourceEffective: []*github.Repository{testRepository("custom", "custom-reader")},
			wantGrants:      []string{"custom:custom-reader"},
			wantWarnings:    []string{"repository custom uses custom role custom-reader; definitions may differ in the destination organization"},
		},
		{
			name:            "incomparable destination parent is skipped only for inherited sources",
			mode:            CopyEffective,
			sameOrg:         true,
			sourceDirect:    []*github.Repository{testRepository("direct", "pull")},
			sourceEffective: []*github.Repository{testRepository("direct", "pull"), testRepository("inherited", "pull")},
			destinationParent: []*github.Repository{
				testRepository("direct", "custom-reader"),
				testRepository("inherited", "custom-reader"),
			},
			wantGrants: []string{"direct:pull"},
			wantWarnings: []string{
				"repository direct has incomparable source permission pull and destination parent permission custom-reader; effective access may differ",
				"repository inherited has incomparable source permission pull and destination parent permission custom-reader; skipped; use --force to override",
			},
		},
		{
			name:                 "repository names are compared without case",
			mode:                 CopyWithInherited,
			sameOrg:              true,
			sourceEffective:      []*github.Repository{testRepository("Sample", "push")},
			destinationEffective: []*github.Repository{testRepository("sample", "pull")},
			destinationDirect:    []*github.Repository{testRepository("SAMPLE", "pull")},
			wantGrants:           []string{"Sample:push"},
		},
		{
			name:                 "effective skips incomparable destination direct custom role masked by parent",
			mode:                 CopyEffective,
			sameOrg:              true,
			sourceEffective:      []*github.Repository{testRepository("sample", "pull")},
			destinationEffective: []*github.Repository{testRepository("sample", "admin")},
			destinationDirect:    []*github.Repository{testRepository("sample", "custom-reader")},
			destinationParent:    []*github.Repository{testRepository("sample", "admin")},
			wantWarnings: []string{
				"repository sample has incomparable source permission pull and destination direct permission custom-reader; skipped; use --force to override",
			},
		},
		{
			name:                 "force overwrites incomparable destination direct custom role masked by parent",
			mode:                 CopyEffective,
			sameOrg:              true,
			force:                true,
			sourceEffective:      []*github.Repository{testRepository("sample", "pull")},
			destinationEffective: []*github.Repository{testRepository("sample", "admin")},
			destinationDirect:    []*github.Repository{testRepository("sample", "custom-reader")},
			destinationParent:    []*github.Repository{testRepository("sample", "admin")},
			wantGrants:           []string{"sample:pull"},
			wantWarnings: []string{
				"repository sample inherits destination parent permission admin, which is stronger than pull; effective access will remain stronger",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selected, warnings, err := selectCopyRepositories(
				tt.mode, tt.sameOrg, tt.force, tt.sourceDirect, tt.sourceEffective, tt.destinationDirect, tt.destinationEffective, tt.destinationParent,
			)
			if err != nil {
				t.Fatalf("selectCopyRepositories() error = %v", err)
			}
			plan := gh.PlanTeamRepoPermissionChanges(selected, tt.destinationDirect, nil, true, false)
			var grants []string
			for _, grant := range plan.Grant {
				grants = append(grants, grant.Repository.GetName()+":"+grant.Permission)
			}
			if !slices.Equal(grants, tt.wantGrants) {
				t.Errorf("grants = %q, want %q", grants, tt.wantGrants)
			}
			if !slices.Equal(warnings, tt.wantWarnings) {
				t.Errorf("warnings = %q, want %q", warnings, tt.wantWarnings)
			}
		})
	}
}

func TestIncludeInheritedStandardPermissionHierarchy(t *testing.T) {
	for sourceRank, source := range gh.PermissionsList {
		for destinationRank, destination := range gh.PermissionsList {
			t.Run(source+"-to-"+destination, func(t *testing.T) {
				selected, warnings, err := selectCopyRepositories(
					CopyWithInherited, true, false, nil,
					[]*github.Repository{testRepository("sample", source)}, nil,
					[]*github.Repository{testRepository("sample", destination)}, nil,
				)
				if err != nil {
					t.Fatal(err)
				}
				want := sourceRank < destinationRank
				if got := len(selected) == 1; got != want {
					t.Errorf("selected = %v, want grant %t", selected, want)
				}
				if len(warnings) != 0 {
					t.Errorf("warnings = %q, want none", warnings)
				}
			})
		}
	}
}

func TestCopyRepositorySelectionRejectsInvalidData(t *testing.T) {
	tests := []struct {
		name            string
		sourceDirect    []*github.Repository
		sourceEffective []*github.Repository
		want            string
	}{
		{"missing name", []*github.Repository{{}}, nil, "source direct repository has no name"},
		{"duplicate name", []*github.Repository{testRepository("sample", "pull"), testRepository("SAMPLE", "push")}, nil, "returned more than once"},
		{"missing role", []*github.Repository{{Name: github.Ptr("sample")}}, nil, "no usable direct permission"},
		{"missing effective source", []*github.Repository{testRepository("sample", "pull")}, nil, "missing from effective permissions"},
		{"missing effective role", nil, []*github.Repository{{Name: github.Ptr("sample")}}, "no usable effective permission"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := selectCopyRepositories(CopyEffective, true, false, tt.sourceDirect, tt.sourceEffective, nil, nil, nil)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestSameOrganizationIncludesHost(t *testing.T) {
	t.Setenv("GH_HOST", "github.example.test")
	tests := []struct {
		source, destination repository.Repository
		want                bool
	}{
		{repository.Repository{Owner: "Example"}, repository.Repository{Host: "github.example.test", Owner: "example"}, true},
		{repository.Repository{Host: "github.example.test", Owner: "example"}, repository.Repository{Host: "other.example.test", Owner: "example"}, false},
		{repository.Repository{Owner: "first"}, repository.Repository{Owner: "second"}, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s-to-%s/%s", tt.source.Host, tt.source.Owner, tt.destination.Host, tt.destination.Owner), func(t *testing.T) {
			if got := sameOrganization(tt.source, tt.destination); got != tt.want {
				t.Errorf("sameOrganization() = %t, want %t", got, tt.want)
			}
		})
	}
}
