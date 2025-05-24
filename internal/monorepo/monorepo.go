// Package monorepo provides functions to work with monorepository configuration.
package monorepo

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

var (
	ErrNoProjects = errors.New("no projects found in configuration file despite operating in monorepo mode")
	ErrNoName     = errors.New("project has no name")
	ErrNoPath     = errors.New("project has no path")
)

type Project struct {
	Path string
	Name string
}

// Unmarshall takes a raw Viper configuration and returns a slice of Project representing various projects in a
// monorepo.
func Unmarshall(input []map[string]string, filter []string) ([]Project, error) {
	if len(input) == 0 {
		return nil, ErrNoProjects
	}

	projects := []Project{}
	missing := slices.Clone(filter)

	for _, p := range input {

		name, ok := p["name"]
		if !ok {
			return nil, ErrNoName
		}

		if len(filter) != 0 && !slices.Contains(filter, name) {
			continue
		}

		path, ok := p["path"]
		if !ok {
			return nil, ErrNoPath
		}

		project := Project{
			Name: name,
			Path: filepath.Clean(path),
		}

		projects = append(projects, project)
		missing = slices.DeleteFunc(missing, func(s string) bool { return s == name })
	}

	if len(missing) != 0 {
		return nil, fmt.Errorf("filtered project(s) `%s` could not be found", strings.Join(missing, ", "))
	}

	return projects, nil
}
