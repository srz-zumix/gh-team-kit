package repocollab

import (
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

func TestBuildRemoveChanges(t *testing.T) {
	users := []*github.User{
		{Login: new("alice"), Permissions: &github.RepositoryPermissions{Pull: new(true)}},
		{Login: new("bob")},
	}
	changes := BuildRemoveChanges("github.com/o/r", users)
	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	for i, want := range []string{"alice", "bob"} {
		c := changes[i]
		if c.Target != want || c.Action != gh.SyncChangeRemove || c.Destination != "github.com/o/r" {
			t.Errorf("unexpected change[%d]: %+v", i, c)
		}
	}
}

func TestBuildRemoveChangesEmpty(t *testing.T) {
	if got := BuildRemoveChanges("github.com/o/r", nil); len(got) != 0 {
		t.Errorf("expected no changes, got %d", len(got))
	}
}
