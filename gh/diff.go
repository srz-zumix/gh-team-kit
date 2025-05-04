package gh

import (
	"fmt"

	"github.com/google/go-github/v71/github"
)

// Diff represents the diff between two repositories.
type Diff struct {
	Left  *github.Repository
	Right *github.Repository
}

func (d *Diff) GetFullName() string {
	if d.Left != nil {
		return *d.Left.FullName
	}
	if d.Right != nil {
		return *d.Right.FullName
	}
	return ""
}

func (d *Diff) GetDiffLines(leftTeamSlug, rightTeamSlug string) string {
	var diff string
	fullName := d.GetFullName()
	leftPerm := GetRepositoryPermissions(d.Left)
	rightPerm := GetRepositoryPermissions(d.Right)
	diff += fmt.Sprintf("diff --gh team-kit repo diff %s %s %s\n", leftTeamSlug, rightTeamSlug, fullName)
	if d.Left != nil && d.Right != nil {
		diff += fmt.Sprintf("--- %s %s\n", *d.Left.FullName, leftTeamSlug)
		diff += fmt.Sprintf("+++ %s %s\n", *d.Right.FullName, rightTeamSlug)
		diff += fmt.Sprintf("- %s\n", leftPerm)
		diff += fmt.Sprintf("+ %s\n", rightPerm)
	} else if d.Left != nil {
		diff += fmt.Sprintf("--- %s %s\n", *d.Left.FullName, leftTeamSlug)
		diff += "+++ /dev/null\n"
		diff += fmt.Sprintf("- %s\n", leftPerm)
	} else if d.Right != nil {
		diff += "--- /dev/null\n"
		diff += fmt.Sprintf("+++ %s %s\n", *d.Right.FullName, rightTeamSlug)
		diff += fmt.Sprintf("+ %s\n", rightPerm)
	}
	return diff
}

type Diffs []Diff

func (d Diffs) Left() []*github.Repository {
	var repos []*github.Repository
	for _, diff := range d {
		if diff.Left != nil {
			repos = append(repos, diff.Left)
		}
	}
	return repos
}

func (d Diffs) Right() []*github.Repository {
	var repos []*github.Repository
	for _, diff := range d {
		if diff.Right != nil {
			repos = append(repos, diff.Right)
		}
	}
	return repos
}

func (d Diffs) GetDiffLines(leftTeamSlug, rightTeamSlug string) string {
	var diffLines string
	for _, diff := range d {
		diffLines += diff.GetDiffLines(leftTeamSlug, rightTeamSlug)
	}
	return diffLines
}

func findRepository(target *github.Repository, repos []*github.Repository) *github.Repository {
	for _, r := range repos {
		if *r.ID == *target.ID {
			return r
		}
	}
	return nil
}

func CompareRepository(left, right *github.Repository) *Diff {
	if GetRepositoryPermissions(left) == GetRepositoryPermissions(right) {
		return nil
	}
	diff := Diff{
		Left:  left,
		Right: right,
	}
	return &diff
}

func CompareRepositories(left, right []*github.Repository) Diffs {
	var diffs []Diff
	for _, l := range left {
		r := findRepository(l, right)
		diff := CompareRepository(l, r)
		if diff != nil {
			diffs = append(diffs, *diff)
		}
	}
	return diffs
}
