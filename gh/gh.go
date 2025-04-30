package gh

import (
	"github.com/cli/go-gh/v2/pkg/repository"
)

// ParseRepository parses a string into a go-gh Repository object. If the string is empty, it returns the current repository.
func ParseRepository(input string) (repository.Repository, error) {
	if input == "" {
		return repository.Current()
	}
	return repository.Parse(input)
}
