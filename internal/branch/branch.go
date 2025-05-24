// Package branch provides functions to handle branch configuration.
package branch

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

var (
	ErrNoBranch = errors.New("no branch configuration")
	ErrNoName   = errors.New("no name in branch configuration")
)

type Branch struct {
	Name       string
	Prerelease bool
}

// Unmarshall takes a raw Viper configuration and returns a slice of Branch representing a branch configuration.
func Unmarshall(input []map[string]any, filter []string) ([]Branch, error) {
	if len(input) == 0 {
		return nil, ErrNoBranch
	}

	branches := []Branch{}
	missing := slices.Clone(filter)

	for _, b := range input {

		name, ok := b["name"]
		if !ok {
			return nil, ErrNoName
		}

		stringName, ok := name.(string)
		if !ok {
			return nil, fmt.Errorf("could not assert that the \"name\" property of the branch configuration is a string")
		}

		if len(filter) != 0 && !slices.Contains(filter, stringName) {
			continue
		}

		branch := Branch{Name: stringName}

		prerelease, ok := b["prerelease"]
		if ok {
			boolPrerelease, ok := prerelease.(bool)
			if !ok {
				return nil, fmt.Errorf("could not assert that the \"prerelease\" property of the branch configuration is a bool")
			}

			branch.Prerelease = boolPrerelease
		}

		branches = append(branches, branch)
		missing = slices.DeleteFunc(missing, func(s string) bool { return s == stringName })
	}

	slog.Info("--->", branches)
	if len(missing) != 0 {
		return nil, fmt.Errorf("filtered branche(s) `%s` could not be found", strings.Join(missing, ", "))
	}

	return branches, nil
}
