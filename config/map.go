package config

// ApplyUserMapping applies the user mapping to an OrganizationConfig.
// It converts all source logins to destination logins in Maintainers, Members, and ExcludedTeamMembers.
func ApplyUserMapping(orgConfig *OrganizationConfig, mapping map[string]string) error {
	if len(mapping) == 0 {
		return nil
	}

	for i := range orgConfig.Teams {
		team := &orgConfig.Teams[i]

		// Apply mapping to Maintainers
		for j, maintainer := range team.Maintainers {
			if dst, exists := mapping[maintainer]; exists {
				team.Maintainers[j] = dst
			}
		}

		// Apply mapping to Members
		for j, member := range team.Members {
			if dst, exists := mapping[member]; exists {
				team.Members[j] = dst
			}
		}

		// Apply mapping to CodeReviewSettings.ExcludedTeamMembers
		if team.CodeReviewSettings != nil {
			for j, excluded := range team.CodeReviewSettings.ExcludedTeamMembers {
				if dst, exists := mapping[excluded]; exists {
					team.CodeReviewSettings.ExcludedTeamMembers[j] = dst
				}
			}
		}
	}

	return nil
}

// ApplyUserMappingFn applies a resolver function to all logins in an OrganizationConfig.
// resolve receives a login and returns the mapped dst login and whether a mapping was found.
func ApplyUserMappingFn(orgConfig *OrganizationConfig, resolve func(string) (string, bool)) error {
	for i := range orgConfig.Teams {
		team := &orgConfig.Teams[i]

		for j, maintainer := range team.Maintainers {
			if dst, ok := resolve(maintainer); ok {
				team.Maintainers[j] = dst
			}
		}

		for j, member := range team.Members {
			if dst, ok := resolve(member); ok {
				team.Members[j] = dst
			}
		}

		if team.CodeReviewSettings != nil {
			for j, excluded := range team.CodeReviewSettings.ExcludedTeamMembers {
				if dst, ok := resolve(excluded); ok {
					team.CodeReviewSettings.ExcludedTeamMembers[j] = dst
				}
			}
		}
	}

	return nil
}
