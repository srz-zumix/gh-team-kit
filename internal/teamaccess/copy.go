package teamaccess

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

type CopyMode int

const (
	CopyDirect CopyMode = iota
	CopyWithInherited
	CopyEffective
)

type Team struct {
	Client     *gh.GitHubClient
	Repository repository.Repository
	Slug       string
}

type CopyPlan struct {
	destination Team
	changes     []gh.SyncChange
	warnings    []string
	skipped     map[string]struct{}
}

type accessState struct {
	sourceDirect         []*github.Repository
	sourceEffective      []*github.Repository
	destinationDirect    []*github.Repository
	destinationEffective []*github.Repository
	destinationParent    []*github.Repository
}

func (p *CopyPlan) Changes() []gh.SyncChange {
	return slices.Clone(p.changes)
}

func (p *CopyPlan) Warnings() []string {
	return slices.Clone(p.warnings)
}

func PlanCopy(ctx context.Context, source, destination Team, mode CopyMode) (*CopyPlan, error) {
	if source.Slug == "" || destination.Slug == "" {
		return nil, fmt.Errorf("source and destination team slugs must not be empty")
	}
	if mode != CopyDirect && mode != CopyWithInherited && mode != CopyEffective {
		return nil, fmt.Errorf("unsupported team repository copy mode %d", mode)
	}

	state, err := loadAccessState(ctx, source, destination, mode != CopyDirect, mode == CopyEffective)
	if err != nil {
		return nil, err
	}
	selected, warnings, err := selectCopyRepositories(
		mode, sameOrganization(source.Repository, destination.Repository),
		state.sourceDirect, state.sourceEffective, state.destinationEffective, state.destinationParent,
	)
	if err != nil {
		return nil, err
	}
	return planGrants(ctx, destination, selected, state.destinationDirect, state.destinationEffective, warnings)
}

func loadAccessState(ctx context.Context, source, destination Team, includeEffective, includeParent bool) (*accessState, error) {
	state := &accessState{}
	var err error
	state.sourceDirect, err = gh.ListTeamRepos(ctx, source.Client, source.Repository, source.Slug, nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list source team's direct repositories: %w", err)
	}
	state.destinationDirect, err = gh.ListTeamRepos(ctx, destination.Client, destination.Repository, destination.Slug, nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to list destination team's direct repositories: %w", err)
	}
	state.destinationEffective, err = gh.ListTeamRepos(ctx, destination.Client, destination.Repository, destination.Slug, nil, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list destination team's repositories: %w", err)
	}

	if includeEffective {
		state.sourceEffective, err = gh.ListTeamRepos(ctx, source.Client, source.Repository, source.Slug, nil, true)
		if err != nil {
			return nil, fmt.Errorf("failed to list source team's effective repositories: %w", err)
		}
	}
	if includeParent {
		state.destinationParent, err = listParentRepositories(ctx, destination)
		if err != nil {
			return nil, err
		}
	}

	return state, nil
}

func planGrants(ctx context.Context, destination Team, selected, destinationDirect, destinationEffective []*github.Repository, warnings []string) (*CopyPlan, error) {
	directByName, err := repositoriesByName("destination direct", destinationDirect)
	if err != nil {
		return nil, err
	}
	effectiveByName, err := repositoriesByName("destination effective", destinationEffective)
	if err != nil {
		return nil, err
	}

	permissionPlan := gh.PlanTeamRepoPermissionChanges(selected, destinationDirect, nil, true, false)
	plan := &CopyPlan{destination: destination, warnings: warnings, skipped: make(map[string]struct{})}
	for _, grant := range permissionPlan.Grant {
		repoName := grant.Repository.GetName()
		repo := destination.Repository
		repo.Name = repoName
		destinationRepo, err := gh.GetRepository(ctx, destination.Client, repo)
		if err != nil {
			if gh.IsHTTPNotFound(err) {
				plan.skipped[strings.ToLower(repoName)] = struct{}{}
				plan.warnings = append(plan.warnings, fmt.Sprintf(
					"repository %s is unavailable in destination organization %s (404); skipped", repoName, repo.Owner,
				))
				continue
			}
			return plan, fmt.Errorf("failed to check destination repository %s/%s: %w", repo.Owner, repoName, err)
		}
		if destinationRepo == nil {
			return plan, fmt.Errorf("destination repository %s/%s was not returned", repo.Owner, repoName)
		}

		key := strings.ToLower(repoName)
		change := gh.SyncChange{
			Destination: teamDestination(destination),
			Action:      gh.SyncChangeAdd,
			Target:      repoName,
			Permission:  grant.Permission,
		}
		if direct := directByName[key]; direct != nil {
			change.Action = gh.SyncChangeUpdate
			change.Current = gh.GetRepositoryPermissions(direct)
		} else if effective := effectiveByName[key]; effective != nil {
			change.Current = gh.GetRepositoryPermissions(effective)
		}
		plan.changes = append(plan.changes, change)
	}
	return plan, nil
}

func ApplyCopy(ctx context.Context, plan *CopyPlan) error {
	if plan == nil {
		return fmt.Errorf("team repository copy plan must not be nil")
	}
	for _, change := range plan.changes {
		repo := plan.destination.Repository
		repo.Name = change.Target
		if err := gh.AddTeamRepo(ctx, plan.destination.Client, repo, plan.destination.Slug, change.Permission); err != nil {
			return fmt.Errorf("failed to grant team %s access to repository %s/%s: %w",
				plan.destination.Slug, repo.Owner, repo.Name, err)
		}
	}
	return nil
}

func listParentRepositories(ctx context.Context, team Team) ([]*github.Repository, error) {
	destinationTeam, err := gh.GetTeamBySlug(ctx, team.Client, team.Repository, team.Slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get destination team: %w", err)
	}
	if destinationTeam == nil {
		return nil, fmt.Errorf("destination team %s was not returned", team.Slug)
	}
	parent := destinationTeam.GetParent()
	if parent == nil {
		return nil, nil
	}
	if parent.GetSlug() == "" {
		return nil, fmt.Errorf("destination team's parent has no slug")
	}
	repos, err := gh.ListTeamRepos(ctx, team.Client, team.Repository, parent.GetSlug(), nil, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list destination parent team's repositories: %w", err)
	}
	return repos, nil
}

func selectCopyRepositories(
	mode CopyMode, sameOrg bool,
	sourceDirect, sourceEffective, destinationEffective, destinationParent []*github.Repository,
) ([]*github.Repository, []string, error) {
	return selectCopyRepositoriesWithSkipped(mode, sameOrg, sourceDirect, sourceEffective, destinationEffective, destinationParent, nil)
}

func selectCopyRepositoriesWithSkipped(
	mode CopyMode, sameOrg bool,
	sourceDirect, sourceEffective, destinationEffective, destinationParent []*github.Repository,
	skipped map[string]struct{},
) ([]*github.Repository, []string, error) {
	directByName, err := repositoriesByName("source direct", sourceDirect)
	if err != nil {
		return nil, nil, err
	}
	effectiveByName, err := repositoriesByName("source effective", sourceEffective)
	if err != nil {
		return nil, nil, err
	}
	destinationByName, err := repositoriesByName("destination effective", destinationEffective)
	if err != nil {
		return nil, nil, err
	}
	parentByName, err := repositoriesByName("destination parent", destinationParent)
	if err != nil {
		return nil, nil, err
	}
	var warnings []string
	for _, repo := range sourceDirect {
		permission := gh.GetRepositoryPermissions(repo)
		if permission == "none" {
			return nil, nil, fmt.Errorf("source repository %s has no usable direct permission", repo.GetName())
		}
		if mode == CopyEffective && effectiveByName[strings.ToLower(repo.GetName())] == nil {
			return nil, nil, fmt.Errorf("source repository %s is missing from effective permissions", repo.GetName())
		}
		if mode != CopyEffective && !sameOrg && isCustomPermission(permission) {
			warnings = append(warnings, customRoleCopyWarning(repo.GetName(), permission))
		}
	}

	var selected []*github.Repository
	if mode != CopyEffective {
		selected = append(selected, sourceDirect...)
	}
	if mode == CopyDirect {
		return selected, warnings, nil
	}

	for _, repo := range sourceEffective {
		name := repo.GetName()
		key := strings.ToLower(name)
		direct := directByName[key] != nil
		if mode == CopyWithInherited && direct {
			continue
		}
		permission := gh.GetRepositoryPermissions(repo)
		if permission == "none" {
			return nil, nil, fmt.Errorf("source repository %s has no usable effective permission", name)
		}

		if !direct {
			if current := destinationByName[key]; current != nil {
				currentPermission := gh.GetRepositoryPermissions(current)
				comparison, comparable := comparePermissions(permission, currentPermission, sameOrg)
				if !comparable {
					if skipped != nil {
						skipped[key] = struct{}{}
					}
					warnings = append(warnings, fmt.Sprintf(
						"repository %s has incomparable source permission %s and destination permission %s; skipped",
						name, permission, currentPermission,
					))
					continue
				}
				if (mode == CopyWithInherited && comparison <= 0) || (mode == CopyEffective && comparison == 0) {
					continue
				}
			}
		}

		if mode == CopyEffective {
			if parent := parentByName[key]; parent != nil {
				parentPermission := gh.GetRepositoryPermissions(parent)
				comparison, comparable := comparePermissions(permission, parentPermission, sameOrg)
				if !comparable {
					if !direct {
						if skipped != nil {
							skipped[key] = struct{}{}
						}
						warnings = append(warnings, fmt.Sprintf(
							"repository %s has incomparable source permission %s and destination parent permission %s; skipped",
							name, permission, parentPermission,
						))
						continue
					}
					warnings = append(warnings, fmt.Sprintf(
						"repository %s has incomparable source permission %s and destination parent permission %s; effective access may differ",
						name, permission, parentPermission,
					))
				} else if comparison < 0 {
					warnings = append(warnings, fmt.Sprintf(
						"repository %s inherits destination parent permission %s, which is stronger than %s; effective access will remain stronger",
						name, parentPermission, permission,
					))
				}
			}
		}
		if !sameOrg && isCustomPermission(permission) {
			warnings = append(warnings, customRoleCopyWarning(name, permission))
		}
		selected = append(selected, repo)
	}
	return selected, warnings, nil
}

func isCustomPermission(permission string) bool {
	return permission != "none" && !slices.Contains(gh.PermissionsList, permission)
}

func customRoleCopyWarning(repo, permission string) string {
	return fmt.Sprintf(
		"repository %s uses custom role %s; definitions may differ in the destination organization",
		repo, permission,
	)
}

func repositoriesByName(description string, repos []*github.Repository) (map[string]*github.Repository, error) {
	byName := make(map[string]*github.Repository, len(repos))
	for _, repo := range repos {
		if repo == nil || repo.GetName() == "" {
			return nil, fmt.Errorf("%s repository has no name", description)
		}
		key := strings.ToLower(repo.GetName())
		if byName[key] != nil {
			return nil, fmt.Errorf("%s repository %s was returned more than once", description, repo.GetName())
		}
		byName[key] = repo
	}
	return byName, nil
}

func comparePermissions(source, destination string, sameOrg bool) (int, bool) {
	sourceRank := slices.Index(gh.PermissionsList, source)
	destinationRank := slices.Index(gh.PermissionsList, destination)
	if sourceRank >= 0 && destinationRank >= 0 {
		switch {
		case sourceRank < destinationRank:
			return 1, true
		case sourceRank > destinationRank:
			return -1, true
		default:
			return 0, true
		}
	}
	if sourceRank < 0 && destinationRank < 0 && source != "none" && source == destination && sameOrg {
		return 0, true
	}
	return 0, false
}

func sameOrganization(source, destination repository.Repository) bool {
	sourceHost, destinationHost := source.Host, destination.Host
	if sourceHost == "" || destinationHost == "" {
		defaultHost, _ := auth.DefaultHost()
		if sourceHost == "" {
			sourceHost = defaultHost
		}
		if destinationHost == "" {
			destinationHost = defaultHost
		}
	}
	return strings.EqualFold(sourceHost, destinationHost) && strings.EqualFold(source.Owner, destination.Owner)
}

func teamDestination(team Team) string {
	destination := team.Repository.Owner + "/" + team.Slug
	if team.Repository.Host != "" {
		destination = team.Repository.Host + "/" + destination
	}
	return destination
}
