package prgraph

import (
	"bytes"
	"context"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/hmarr/codeowners"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// codeownersLocations lists the standard locations of a CODEOWNERS file in a repository.
var codeownersLocations = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}

// fetchCodeowners retrieves and parses the CODEOWNERS file from one of the
// standard locations in the repository. It returns nil if no file exists.
func fetchCodeowners(ctx context.Context, g *gh.GitHubClient, repo repository.Repository) codeowners.Ruleset {
	for _, path := range codeownersLocations {
		content, err := gh.GetFileContent(ctx, g, repo, path, nil)
		if err != nil {
			continue
		}
		ruleset, err := codeowners.ParseFile(bytes.NewReader(content))
		if err != nil {
			continue
		}
		return ruleset
	}
	return nil
}

// codeownersOwnerNode adds a graph node for a CODEOWNERS owner entry.
// Team owners belonging to the repository owner are shortened to the team slug.
func codeownersOwnerNode(graph *Graph, repo repository.Repository, owner codeowners.Owner) *Node {
	switch owner.Type {
	case codeowners.TeamOwner:
		return graph.AddNode(NodeTypeTeam, trimTeamOrg(owner.Value, repo.Owner))
	default:
		return graph.AddNode(NodeTypeUser, owner.Value)
	}
}

// trimTeamOrg strips the "org/" prefix from a team value when it matches the
// repository owner so that CODEOWNERS teams and org teams share the same node.
func trimTeamOrg(value string, owner string) string {
	prefix := owner + "/"
	if len(value) > len(prefix) && strings.EqualFold(value[:len(prefix)], prefix) {
		return value[len(prefix):]
	}
	return value
}
