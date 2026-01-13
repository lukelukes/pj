package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"pj/internal/catalog"
	"strings"
)

type AmbiguousMatchError struct {
	Query   string
	Matches []catalog.Project
}

func (e *AmbiguousMatchError) Error() string {
	return fmt.Sprintf("multiple projects match %q", e.Query)
}

func (e *AmbiguousMatchError) WriteMatches(w io.Writer) {
	fmt.Fprintln(w, "Multiple projects match. Please be more specific:")
	for _, p := range e.Matches {
		fmt.Fprintf(w, "  - %s (%s)\n", p.Name, p.Path)
	}
}

func handleFindError(w io.Writer, err error) bool {
	var ambErr *AmbiguousMatchError
	if errors.As(err, &ambErr) {
		ambErr.WriteMatches(w)
		return true
	}
	return false
}

func findProject(cat catalog.Catalog, query string) (catalog.Project, error) {
	projects := cat.Search(query)
	if len(projects) == 0 {
		return catalog.Project{}, fmt.Errorf("no project found matching: %s", query)
	}
	if len(projects) > 1 {
		return catalog.Project{}, &AmbiguousMatchError{Query: query, Matches: projects}
	}
	return projects[0], nil
}

func splitCommand(s string) []string {
	var result []string
	var current strings.Builder
	var inQuote rune

	for _, r := range s {
		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}
		switch r {
		case '"', '\'':
			inQuote = r
		case ' ', '\t':
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func resolveEditor(project catalog.Project) ([]string, error) {
	editor := project.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vim"
	}

	parts := splitCommand(editor)
	if len(parts) == 0 {
		return nil, fmt.Errorf("editor %q is empty after parsing", editor)
	}

	if _, err := exec.LookPath(parts[0]); err != nil {
		return nil, fmt.Errorf("editor %q not found in PATH", parts[0])
	}
	return parts, nil
}
