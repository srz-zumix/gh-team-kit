package gh

import (
	"fmt"
	"maps"
	"slices"

	"github.com/google/go-github/v71/github"
)

// UnionUsers calculates the union of two slices of *github.User
func UnionUsers(users1, users2 []*github.User) []*github.User {
	userMap := make(map[int64]*github.User)

	for _, user := range users1 {
		userMap[user.GetID()] = user
	}

	for _, user := range users2 {
		userMap[user.GetID()] = user
	}

	return slices.Collect(maps.Values(userMap))
}

// IntersectionUsers calculates the intersection of two slices of *github.User
func IntersectionUsers(users1, users2 []*github.User) []*github.User {
	userMap := make(map[int64]*github.User)

	// Add all users from the first slice to the map
	for _, user := range users1 {
		userMap[user.GetID()] = user
	}

	result := []*github.User{}

	// Check for common users in the second slice
	for _, user := range users2 {
		if _, exists := userMap[user.GetID()]; exists {
			result = append(result, user)
		}
	}

	return result
}

// DifferenceUsers calculates the difference of two slices of *github.User
func DifferenceUsers(users1, users2 []*github.User) []*github.User {
	userMap := make(map[int64]*github.User)

	// Add all users from the first slice to the map
	for _, user := range users1 {
		userMap[user.GetID()] = user
	}

	// Remove users found in the second slice from the map
	for _, user := range users2 {
		delete(userMap, user.GetID())
	}

	return slices.Collect(maps.Values(userMap))
}

// PerformSetOperation performs a set operation (+, *, -) on two slices of *github.User
func PerformSetOperation(users1, users2 []*github.User, operation string) ([]*github.User, error) {
	switch operation {
	case "+":
		return UnionUsers(users1, users2), nil
	case "*":
		return IntersectionUsers(users1, users2), nil
	case "-":
		return DifferenceUsers(users1, users2), nil
	default:
		return nil, fmt.Errorf("invalid operation: %s, must be one of +, *, -", operation)
	}
}
