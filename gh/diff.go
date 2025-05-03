package gh

import (
	"github.com/google/go-github/v71/github"
)

// Diff represents the diff between two repositories.
type Diff struct {
	Left  *github.Repository
	Right *github.Repository
}

func (d *Diff) GetDiff() string {
	var diff string
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
