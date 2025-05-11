package gh

import (
	"fmt"
	"maps"
	"slices"
	"strings"

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

// SymmetricDifferenceUsers calculates the symmetric difference of two slices of *github.User
func SymmetricDifferenceUsers(users1, users2 []*github.User) []*github.User {
	userMap1 := make(map[int64]*github.User)
	userMap2 := make(map[int64]*github.User)

	// Add all users from the first slice to the first map
	for _, user := range users1 {
		userMap1[user.GetID()] = user
	}

	// Add all users from the second slice to the second map
	for _, user := range users2 {
		userMap2[user.GetID()] = user
	}

	result := []*github.User{}

	// Add users that are in users1 but not in users2
	for _, user := range users1 {
		if _, exists := userMap2[user.GetID()]; !exists {
			result = append(result, user)
		}
	}

	// Add users that are in users2 but not in users1
	for _, user := range users2 {
		if _, exists := userMap1[user.GetID()]; !exists {
			result = append(result, user)
		}
	}

	return result
}

type SetsOperationFunc func(users1, users2 []*github.User) []*github.User

var setOperationFuncMap = map[string]SetsOperationFunc{
	"|": UnionUsers,
	"&": IntersectionUsers,
	"-": DifferenceUsers,
	"^": SymmetricDifferenceUsers,
}

func GetSetsOperationFunc(operation string) (SetsOperationFunc, error) {
	if setOperationFuncMap[operation] == nil {
		return nil, fmt.Errorf("unsupported operation: %s, valid operation are: {%s}", operation, strings.Join(GetSetsOperationKeys(), ","))
	}
	return setOperationFuncMap[operation], nil
}

func GetSetsOperationKeys() []string {
	return slices.Collect(maps.Keys(setOperationFuncMap))
}

// PerformSetOperation performs a set operation (+, *, -) on two slices of *github.User
func PerformSetOperation(users1, users2 []*github.User, operation string) ([]*github.User, error) {
	operationFunc, err := GetSetsOperationFunc(operation)
	if err != nil {
		return nil, err
	}
	return operationFunc(users1, users2), nil
}
