// Package userfilter provides shared helpers for the user listing commands that
// need to resolve suspension state and user details.
package userfilter

import (
	"context"

	"github.com/srz-zumix/go-gh-extension/pkg/cmdflags"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// FilterSuspended applies the suspension filter selected by the flag.
func FilterSuspended(suspended cmdflags.MutuallyExclusiveBoolFlags, users []*gh.GitHubUser) []*gh.GitHubUser {
	if suspended.IsEnabled() {
		return gh.CollectSuspendedUsers(users)
	}
	if suspended.IsDisabled() {
		return gh.ExcludeSuspendedUsers(users)
	}
	return users
}

// Update fetches user information required by the requested output.
// When details are explicitly requested it fetches the full user details,
// otherwise it only resolves the suspension state, which is cheaper.
func Update(ctx context.Context, client *gh.GitHubClient, users []*gh.GitHubUser, fetchDetails bool) ([]*gh.GitHubUser, error) {
	if fetchDetails {
		return gh.UpdateUsers(ctx, client, users)
	}
	return gh.UpdateUsersForSuspension(ctx, client, users)
}

// UpdateAndFilter resolves the information needed for the suspension filter,
// applies it, and then fetches the full details of the remaining users when the
// first fetch only resolved the suspension state and details are rendered.
func UpdateAndFilter(ctx context.Context, client *gh.GitHubClient, users []*gh.GitHubUser, suspended cmdflags.MutuallyExclusiveBoolFlags, fetchDetails, nameOnly bool) ([]*gh.GitHubUser, error) {
	users, err := Update(ctx, client, users, fetchDetails)
	if err != nil {
		return nil, err
	}
	users = FilterSuspended(suspended, users)
	if NeedsDetailsAfterFilter(suspended, fetchDetails, nameOnly) {
		return gh.UpdateUsers(ctx, client, users)
	}
	return users, nil
}

// NeedsDetailsAfterFilter reports whether full user details must be fetched
// after the suspension filter has been applied.
func NeedsDetailsAfterFilter(suspended cmdflags.MutuallyExclusiveBoolFlags, fetchDetails, nameOnly bool) bool {
	return suspended.IsSet() && !fetchDetails && !nameOnly
}
