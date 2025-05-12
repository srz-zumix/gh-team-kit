package gh

import (
	"strings"
	"testing"

	"github.com/google/go-github/v71/github"
)

func TestUnionUsers(t *testing.T) {
	user1 := &github.User{ID: github.Ptr(int64(1))}
	user2 := &github.User{ID: github.Ptr(int64(2))}
	user3 := &github.User{ID: github.Ptr(int64(3))}

	result := UnionUsers([]*github.User{user1, user2}, []*github.User{user2, user3})

	if len(result) != 3 {
		t.Errorf("expected 3 users, got %d", len(result))
	}
	if !containsUser(result, user1) || !containsUser(result, user2) || !containsUser(result, user3) {
		t.Errorf("result does not contain expected users")
	}
}

func TestIntersectionUsers(t *testing.T) {
	user1 := &github.User{ID: github.Ptr(int64(1))}
	user2 := &github.User{ID: github.Ptr(int64(2))}
	user3 := &github.User{ID: github.Ptr(int64(3))}

	result := IntersectionUsers([]*github.User{user1, user2}, []*github.User{user2, user3})

	if len(result) != 1 {
		t.Errorf("expected 1 user, got %d", len(result))
	}
	if !containsUser(result, user2) {
		t.Errorf("result does not contain expected user")
	}
}

func TestDifferenceUsers(t *testing.T) {
	user1 := &github.User{ID: github.Ptr(int64(1))}
	user2 := &github.User{ID: github.Ptr(int64(2))}
	user3 := &github.User{ID: github.Ptr(int64(3))}

	result := DifferenceUsers([]*github.User{user1, user2}, []*github.User{user2, user3})

	if len(result) != 1 {
		t.Errorf("expected 1 user, got %d", len(result))
	}
	if !containsUser(result, user1) {
		t.Errorf("result does not contain expected user")
	}
}

func TestSymmetricDifferenceUsers(t *testing.T) {
	user1 := &github.User{ID: github.Ptr(int64(1))}
	user2 := &github.User{ID: github.Ptr(int64(2))}
	user3 := &github.User{ID: github.Ptr(int64(3))}

	result := SymmetricDifferenceUsers([]*github.User{user1, user2}, []*github.User{user2, user3})

	if len(result) != 2 {
		t.Errorf("expected 2 users, got %d", len(result))
	}
	if !containsUser(result, user1) || !containsUser(result, user3) {
		t.Errorf("result does not contain expected users")
	}
}

func TestPerformSetOperation(t *testing.T) {
	user1 := &github.User{ID: github.Ptr(int64(1))}
	user2 := &github.User{ID: github.Ptr(int64(2))}
	user3 := &github.User{ID: github.Ptr(int64(3))}

	tests := []struct {
		name      string
		operation string
		expected  []*github.User
	}{
		{
			name:      "Union",
			operation: "|",
			expected:  []*github.User{user1, user2, user3},
		},
		{
			name:      "Intersection",
			operation: "&",
			expected:  []*github.User{user2},
		},
		{
			name:      "Difference",
			operation: "-",
			expected:  []*github.User{user1},
		},
		{
			name:      "SymmetricDifference",
			operation: "^",
			expected:  []*github.User{user1, user3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PerformSetOperation([]*github.User{user1, user2}, []*github.User{user2, user3}, tt.operation)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(result) != len(tt.expected) || !containsAllUsers(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetSetsOperationFunc_InvalidOperation(t *testing.T) {
	_, err := GetSetsOperationFunc("invalid")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "unsupported operation") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func containsUser(users []*github.User, user *github.User) bool {
	for _, u := range users {
		if u.GetID() == user.GetID() {
			return true
		}
	}
	return false
}

func containsAllUsers(users, expected []*github.User) bool {
	for _, e := range expected {
		if !containsUser(users, e) {
			return false
		}
	}
	return true
}
