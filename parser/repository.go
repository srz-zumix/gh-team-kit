package parser

import (
	"errors"

	"github.com/cli/go-gh/v2/pkg/repository"
)

type RepositoryOption func(*repository.Repository) error

func RepositoryInput(input string) RepositoryOption {
	return func(r *repository.Repository) error {
		if input == "" {
			return nil
		}
		repo, err := repository.Parse(input)
		if err != nil {
			return err
		}
		if r.Host != "" && r.Host != repo.Host {
			return errors.New("conflicting host")
		}
		if r.Name != "" && r.Name != repo.Name {
			return errors.New("conflicting name")
		}
		if r.Owner != "" && r.Owner != repo.Owner {
			return errors.New("conflicting owner")
		}
		r.Host = repo.Host
		r.Name = repo.Name
		r.Owner = repo.Owner
		return nil
	}
}

func RepositoryOwner(input string) RepositoryOption {
	return func(r *repository.Repository) error {
		if input == "" {
			return nil
		}
		if r.Owner != "" && r.Owner != input {
			return errors.New("conflicting owner")
		}
		r.Owner = input
		return nil
	}
}

// ParseRepository parses a string into a go-gh Repository object. If the string is empty, it returns the current repository.
func Repository(opts ...RepositoryOption) (repository.Repository, error) {
	var r repository.Repository
	for _, opt := range opts {
		if err := opt(&r); err != nil {
			return repository.Repository{}, err
		}
	}
	if r.Host == "" && r.Name == "" && r.Owner == "" {
		return repository.Current()
	}
	return r, nil
}
