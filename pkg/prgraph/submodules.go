package prgraph

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// fetchSubmodulePaths retrieves the .gitmodules file from the repository and
// returns the set of registered submodule paths. It returns nil when the
// repository has no .gitmodules file.
func fetchSubmodulePaths(ctx context.Context, g *gh.GitHubClient, repo repository.Repository) map[string]bool {
	content, err := gh.GetFileContent(ctx, g, repo, ".gitmodules", nil)
	if err != nil {
		return nil
	}
	return parseSubmodulePaths(content)
}

// parseSubmodulePaths extracts the "path" entries from .gitmodules content.
func parseSubmodulePaths(content []byte) map[string]bool {
	paths := make(map[string]bool)
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok || strings.TrimSpace(key) != "path" {
			continue
		}
		if value = strings.Trim(strings.TrimSpace(value), `"`); value != "" {
			paths[strings.Trim(value, "/")] = true
		}
	}
	return paths
}
