package prgraph

import (
	"context"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
	"github.com/srz-zumix/go-gh-extension/pkg/logger"
)

// fetchTrackedPaths retrieves the tracked paths of the repository's default
// branch (blobs and submodule commit entries), used to drop paths that no
// longer exist. It returns nil when the tree cannot be resolved or GitHub
// truncated it, because filtering against a partial tree would silently discard
// files that do still exist.
func fetchTrackedPaths(ctx context.Context, g *gh.GitHubClient, repo repository.Repository) map[string]bool {
	repoInfo, err := gh.GetRepository(ctx, g, repo)
	if err != nil {
		logger.Warn("failed to resolve the default branch; --exclude-deleted is disabled",
			"repository", repo.Owner+"/"+repo.Name,
			"error", err,
		)
		return nil
	}
	branch := repoInfo.GetDefaultBranch()
	if branch == "" {
		logger.Warn("repository has no default branch; --exclude-deleted is disabled",
			"repository", repo.Owner+"/"+repo.Name,
		)
		return nil
	}
	tree, err := g.GetGitTree(ctx, repo.Owner, repo.Name, branch, true)
	if err != nil {
		logger.Warn("failed to read the repository tree; --exclude-deleted is disabled",
			"repository", repo.Owner+"/"+repo.Name,
			"branch", branch,
			"error", err,
		)
		return nil
	}
	if tree.GetTruncated() {
		logger.Warn("repository tree is truncated; --exclude-deleted is disabled",
			"repository", repo.Owner+"/"+repo.Name,
			"branch", branch,
		)
		return nil
	}
	paths := make(map[string]bool, len(tree.Entries))
	for _, entry := range tree.Entries {
		// Blobs are files; commit entries are gitlink submodules. Both are
		// tracked paths that pr-graph can emit as nodes.
		switch entry.GetType() {
		case "blob", "commit":
			paths[entry.GetPath()] = true
		}
	}
	return paths
}
