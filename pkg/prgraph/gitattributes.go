package prgraph

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/srz-zumix/go-gh-extension/pkg/gh"
)

// fetchGeneratedMatcher retrieves the .gitattributes file from the repository
// and compiles the path patterns marked with the linguist-generated attribute.
// It returns nil when the repository has no such patterns.
func fetchGeneratedMatcher(ctx context.Context, g *gh.GitHubClient, repo repository.Repository) *ignore.GitIgnore {
	content, err := gh.GetFileContent(ctx, g, repo, ".gitattributes", nil)
	if err != nil {
		return nil
	}
	patterns := parseGeneratedPatterns(content)
	if len(patterns) == 0 {
		return nil
	}
	return ignore.CompileIgnoreLines(patterns...)
}

// parseGeneratedPatterns extracts the path patterns of .gitattributes entries
// setting linguist-generated. Entries unsetting it are returned as negated
// ".gitignore"-style patterns so that later rules override earlier ones.
func parseGeneratedPatterns(content []byte) []string {
	var patterns []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pattern, attrs, ok := cutGitAttributesLine(line)
		if !ok {
			continue
		}
		for _, attr := range attrs {
			switch attr {
			case "linguist-generated", "linguist-generated=true":
				patterns = append(patterns, pattern)
			case "-linguist-generated", "linguist-generated=false":
				patterns = append(patterns, "!"+pattern)
			}
		}
	}
	return patterns
}

// cutGitAttributesLine splits a .gitattributes line into its path pattern and
// attribute list, handling the double-quoted pattern form.
func cutGitAttributesLine(line string) (string, []string, bool) {
	if strings.HasPrefix(line, `"`) {
		end := strings.Index(line[1:], `"`)
		if end < 0 {
			return "", nil, false
		}
		return line[1 : end+1], strings.Fields(line[end+2:]), true
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", nil, false
	}
	return fields[0], fields[1:], true
}
