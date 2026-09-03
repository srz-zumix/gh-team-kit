package prgraph

import (
	"strings"
)

// ParseUserAllowlist extracts the GitHub logins of an allowlist file. Blank
// lines and "#" comments are ignored, a single leading "@" is stripped, and
// duplicates are removed while preserving the order of first appearance.
func ParseUserAllowlist(content []byte) []string {
	var logins []string
	seen := make(map[string]bool)
	for _, raw := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		login := strings.TrimSpace(strings.TrimPrefix(line, "@"))
		if login == "" {
			continue
		}
		key := strings.ToLower(login)
		if seen[key] {
			continue
		}
		seen[key] = true
		logins = append(logins, login)
	}
	return logins
}
