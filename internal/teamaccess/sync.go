package teamaccess

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

type SyncPlan struct {
	grants         *CopyPlan
	removals       []gh.SyncChange
	expected       map[string]*github.Repository
	expectedDirect map[string]string
	expectedAbsent map[string]string
	skipped        map[string]struct{}
	sourceDirect   map[string]*github.Repository
	predicted      []string
	mode           CopyMode
	sameOrg        bool
	complete       bool
}

func (p *SyncPlan) Changes() []gh.SyncChange {
	return append(p.grants.Changes(), p.removals...)
}

func (p *SyncPlan) Warnings() []string {
	return append(p.grants.Warnings(), p.predicted...)
}

func PlanSync(ctx context.Context, source, destination Team, mode CopyMode, force bool) (*SyncPlan, error) {
	if source.Slug == "" || destination.Slug == "" {
		return nil, fmt.Errorf("source and destination team slugs must not be empty")
	}
	if mode != CopyDirect && mode != CopyWithInherited && mode != CopyEffective {
		return nil, fmt.Errorf("unsupported team repository sync mode %d", mode)
	}
	state, err := loadAccessState(ctx, source, destination, mode != CopyDirect, true)
	if err != nil {
		return nil, err
	}
	if mode == CopyDirect {
		state.sourceEffective = state.sourceDirect
	}
	sourceDirect, err := repositoriesByName("source direct", state.sourceDirect)
	if err != nil {
		return nil, err
	}
	sourceEffective, err := repositoriesByName("source effective", state.sourceEffective)
	if err != nil {
		return nil, err
	}
	destinationDirect, err := repositoriesByName("destination direct", state.destinationDirect)
	if err != nil {
		return nil, err
	}
	destinationEffective, err := repositoriesByName("destination effective", state.destinationEffective)
	if err != nil {
		return nil, err
	}
	parent, err := repositoriesByName("destination parent", state.destinationParent)
	if err != nil {
		return nil, err
	}
	for _, key := range sortedRepositoryKeys(sourceDirect) {
		repo := sourceDirect[key]
		if sourceEffective[key] == nil {
			return nil, fmt.Errorf("source repository %s is missing from effective permissions", repo.GetName())
		}
	}

	sameOrg := sameOrganization(source.Repository, destination.Repository)
	skipped := make(map[string]struct{})
	redundant := make(map[string]string)
	var selected []*github.Repository
	var warnings []string
	if mode == CopyEffective {
		selected, warnings, err = selectCopyRepositoriesWithSkipped(
			mode, sameOrg, force, state.sourceDirect, state.sourceEffective,
			state.destinationDirect, state.destinationEffective, state.destinationParent, skipped,
		)
	} else {
		selected, warnings, err = selectCopyRepositories(CopyDirect, sameOrg, force, state.sourceDirect, nil, nil, nil, nil)
		if err == nil && mode == CopyWithInherited {
			selected, warnings, err = selectInheritedSync(
				selected, warnings, state.sourceEffective, sourceDirect, destinationDirect, parent,
				sameOrg, force, skipped, redundant,
			)
		}
	}
	if err != nil {
		return nil, err
	}
	grantPlan, planErr := planGrants(ctx, destination, selected, state.destinationDirect, state.destinationEffective, warnings, sameOrg, force)
	if grantPlan == nil {
		return nil, planErr
	}
	plan := &SyncPlan{
		grants: grantPlan, expected: sourceEffective, sourceDirect: sourceDirect,
		expectedDirect: make(map[string]string), expectedAbsent: make(map[string]string),
		skipped: skipped, mode: mode, sameOrg: sameOrg,
	}
	for key := range grantPlan.skipped {
		plan.skipped[key] = struct{}{}
	}
	if planErr != nil {
		return plan, planErr
	}

	for _, repo := range selected {
		key := strings.ToLower(repo.GetName())
		if _, skip := skipped[key]; !skip {
			plan.expectedDirect[key] = gh.GetRepositoryPermissions(repo)
		}
	}
	for _, repo := range state.destinationDirect {
		key := strings.ToLower(repo.GetName())
		if _, skip := skipped[key]; skip {
			continue
		}
		_, removeRedundant := redundant[key]
		if sourceEffective[key] != nil && !removeRedundant {
			continue
		}
		if !force && !safeToRemoveDirectPermission(repo, destinationEffective[key], parent[key]) {
			skipped[key] = struct{}{}
			current := "unknown"
			if destinationEffective[key] != nil {
				current = gh.GetRepositoryPermissions(destinationEffective[key])
			}
			plan.grants.warnings = append(plan.grants.warnings, fmt.Sprintf(
				"repository %s cannot safely remove direct permission %s with effective permission %s and parent permission %s; skipped; use --force to override",
				repo.GetName(), gh.GetRepositoryPermissions(repo), current, gh.GetRepositoryPermissions(parent[key]),
			))
			continue
		}
		plan.expectedAbsent[key] = repo.GetName()
		plan.removals = append(plan.removals, gh.SyncChange{
			Destination: teamDestination(destination),
			Action:      gh.SyncChangeRemove,
			Target:      repo.GetName(),
			Current:     gh.GetRepositoryPermissions(repo),
		})
	}
	for _, repo := range state.destinationParent {
		key := strings.ToLower(repo.GetName())
		if _, skip := skipped[key]; skip {
			continue
		}
		if sourceEffective[key] == nil {
			plan.predicted = append(plan.predicted, fmt.Sprintf("repository %s remains accessible through an inherited permission", repo.GetName()))
		} else if mode != CopyEffective {
			direct := sourceDirect[key]
			if direct != nil {
				comparison, comparable := comparePermissions(gh.GetRepositoryPermissions(direct), gh.GetRepositoryPermissions(repo), sameOrg)
				if !comparable || comparison < 0 {
					plan.predicted = append(plan.predicted, fmt.Sprintf(
						"repository %s has an inherited permission that may differ from the synchronized direct permission",
						repo.GetName(),
					))
				}
			}
		}
	}
	plan.complete = true
	return plan, nil
}

func safeToRemoveDirectPermission(direct, effective, parent *github.Repository) bool {
	if direct == nil || effective == nil || parent == nil {
		return false
	}
	directPermission := gh.GetRepositoryPermissions(direct)
	effectivePermission := gh.GetRepositoryPermissions(effective)
	parentPermission := gh.GetRepositoryPermissions(parent)
	directComparison, directComparable := comparePermissions(directPermission, parentPermission, true)
	effectiveComparison, effectiveComparable := comparePermissions(effectivePermission, parentPermission, true)
	return directComparable && effectiveComparable && directComparison <= 0 && effectiveComparison <= 0
}

func selectInheritedSync(
	selected []*github.Repository, warnings []string, sourceEffective []*github.Repository,
	sourceDirect, destinationDirect, parent map[string]*github.Repository,
	sameOrg, force bool, skipped map[string]struct{}, redundant map[string]string,
) ([]*github.Repository, []string, error) {
	for _, repo := range sourceEffective {
		name := repo.GetName()
		key := strings.ToLower(name)
		if sourceDirect[key] != nil {
			continue
		}
		permission := gh.GetRepositoryPermissions(repo)
		if permission == "none" {
			return nil, nil, fmt.Errorf("source repository %s has no usable effective permission", name)
		}
		if direct := destinationDirect[key]; direct != nil {
			current := gh.GetRepositoryPermissions(direct)
			if _, comparable := comparePermissions(permission, current, sameOrg); !comparable && !force {
				warnings = append(warnings, incomparablePermissionSkipWarning(name, permission, current, "direct"))
				skipped[key] = struct{}{}
				continue
			}
		}
		if inherited := parent[key]; inherited != nil {
			parentPermission := gh.GetRepositoryPermissions(inherited)
			comparison, comparable := comparePermissions(permission, parentPermission, sameOrg)
			if !comparable {
				if !force {
					warnings = append(warnings, incomparablePermissionSkipWarning(name, permission, parentPermission, "parent"))
					skipped[key] = struct{}{}
					continue
				}
				warnings = append(warnings, fmt.Sprintf(
					"repository %s has incomparable source permission %s and destination parent permission %s; effective access may differ",
					name, permission, parentPermission,
				))
			} else if comparison <= 0 {
				redundant[key] = name
				continue
			}
		}
		if !sameOrg && isCustomPermission(permission) {
			warnings = append(warnings, customRoleCopyWarning(name, permission))
		}
		selected = append(selected, repo)
	}
	return selected, warnings, nil
}

func ApplySyncGrants(ctx context.Context, plan *SyncPlan) error {
	if plan == nil {
		return fmt.Errorf("team repository sync plan must not be nil")
	}
	return ApplyCopy(ctx, plan.grants)
}

func ApplySync(ctx context.Context, plan *SyncPlan) ([]string, error) {
	if plan == nil || !plan.complete {
		return nil, fmt.Errorf("team repository sync plan is incomplete")
	}
	warnings := plan.grants.Warnings()
	if err := ApplySyncGrants(ctx, plan); err != nil {
		return warnings, err
	}
	destination := plan.grants.destination
	for _, change := range plan.removals {
		repo := destination.Repository
		repo.Name = change.Target
		if err := gh.RemoveTeamRepo(ctx, destination.Client, repo, destination.Slug); err != nil {
			return warnings, fmt.Errorf("failed to revoke team %s access to repository %s/%s: %w",
				destination.Slug, repo.Owner, repo.Name, err)
		}
	}
	direct, err := gh.ListTeamRepos(ctx, destination.Client, destination.Repository, destination.Slug, nil, false)
	if err != nil {
		return warnings, fmt.Errorf("failed to verify destination team's direct permissions after sync; changes may already have been applied: %w", err)
	}
	effective, err := gh.ListTeamRepos(ctx, destination.Client, destination.Repository, destination.Slug, nil, true)
	if err != nil {
		return warnings, fmt.Errorf("failed to verify destination team's effective permissions after sync; changes may already have been applied: %w", err)
	}
	verified, err := plan.verify(direct, effective)
	if err != nil {
		return warnings, fmt.Errorf("failed to verify destination team permissions after sync; changes may already have been applied: %w", err)
	}
	return append(warnings, verified...), nil
}

func (p *SyncPlan) verify(direct, effective []*github.Repository) ([]string, error) {
	directByName, err := repositoriesByName("destination direct", direct)
	if err != nil {
		return nil, err
	}
	effectiveByName, err := repositoriesByName("destination effective", effective)
	if err != nil {
		return nil, err
	}
	var warnings []string
	for _, key := range slices.Sorted(maps.Keys(p.expectedDirect)) {
		permission := p.expectedDirect[key]
		if _, skip := p.skipped[key]; skip {
			continue
		}
		actual := gh.GetRepositoryPermissions(directByName[key])
		if actual != permission {
			warnings = append(warnings, fmt.Sprintf(
				"repository %s has direct permission %s after synchronization; expected %s", p.expected[key].GetName(), actual, permission,
			))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(p.expectedAbsent)) {
		if _, skip := p.skipped[key]; skip {
			continue
		}
		name := p.expectedAbsent[key]
		if directByName[key] != nil {
			warnings = append(warnings, fmt.Sprintf("repository %s still has direct access after synchronization", name))
		}
	}
	for _, key := range sortedRepositoryKeys(p.expected) {
		if _, skip := p.skipped[key]; skip {
			continue
		}
		expected := p.expected[key]
		if p.mode == CopyWithInherited && p.sourceDirect[key] != nil {
			expected = p.sourceDirect[key]
		}
		actual := effectiveByName[key]
		if actual == nil {
			warnings = append(warnings, fmt.Sprintf("repository %s is missing from destination effective access after synchronization", expected.GetName()))
			continue
		}
		expectedPermission := gh.GetRepositoryPermissions(expected)
		actualPermission := gh.GetRepositoryPermissions(actual)
		comparison, comparable := comparePermissions(expectedPermission, actualPermission, p.sameOrg)
		if !comparable {
			warnings = append(warnings, fmt.Sprintf(
				"repository %s has incomparable source permission %s and effective destination permission %s after synchronization",
				expected.GetName(), expectedPermission, actualPermission,
			))
		} else if comparison != 0 && (p.mode == CopyEffective || p.sourceDirect[key] != nil || comparison > 0) {
			warnings = append(warnings, fmt.Sprintf(
				"repository %s has effective permission %s after synchronization; expected %s",
				expected.GetName(), actualPermission, expectedPermission,
			))
		}
	}
	for _, repo := range effective {
		key := strings.ToLower(repo.GetName())
		if _, skip := p.skipped[key]; skip {
			continue
		}
		if p.expected[key] == nil {
			warnings = append(warnings, fmt.Sprintf("repository %s remains accessible after synchronization", repo.GetName()))
		}
	}
	return warnings, nil
}

func sortedRepositoryKeys(repos map[string]*github.Repository) []string {
	keys := make([]string, 0, len(repos))
	for key := range repos {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
