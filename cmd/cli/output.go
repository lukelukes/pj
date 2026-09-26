package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"pj/cmd/cli/render"
	"pj/internal/catalog"
	"slices"
	"time"
)

type Output string

const (
	OutputTable Output = "table"
	OutputNames Output = "names"
	OutputPaths Output = "paths"
	OutputJSON  Output = "json"
)

type projectJSON struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Description  string    `json:"description"`
	Editor       string    `json:"editor"`
	AddedAt      time.Time `json:"added_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

func printProjects(w io.Writer, r render.Renderer, projects []catalog.Project, output Output) error {
	switch output {
	case OutputTable:
		return printTable(w, r, projects)
	case OutputNames:
		return printLines(w, byName(projects), func(p catalog.Project) string { return p.Name })
	case OutputPaths:
		return printLines(w, byName(projects), func(p catalog.Project) string { return p.Path })
	case OutputJSON:
		return printJSON(w, byName(projects))
	default:
		return fmt.Errorf("unknown output %q (valid outputs: table, names, paths, json)", output)
	}
}

func compareByName(a, b catalog.Project) int {
	return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Path, b.Path), cmp.Compare(a.ID, b.ID))
}

func byName(projects []catalog.Project) []catalog.Project {
	return slices.SortedFunc(slices.Values(projects), compareByName)
}

func printTable(w io.Writer, r render.Renderer, projects []catalog.Project) error {
	items := make([]render.ProjectListItem, len(projects))
	for i, p := range projects {
		items[i] = render.ProjectListItem{
			Name:        p.Name,
			Path:        p.Path,
			Description: p.Description,
			Timestamp:   getMtime(p.Path),
		}
	}
	slices.SortFunc(items, func(a, b render.ProjectListItem) int {
		return cmp.Or(b.Timestamp.Compare(a.Timestamp), cmp.Compare(a.Name, b.Name), cmp.Compare(a.Path, b.Path))
	})
	_, err := fmt.Fprint(w, r.RenderProjectList(render.ProjectListView{Items: items}))
	return err
}

func getMtime(path string) time.Time {
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

func printLines(w io.Writer, projects []catalog.Project, value func(catalog.Project) string) error {
	for _, p := range projects {
		if _, err := fmt.Fprintln(w, value(p)); err != nil {
			return err
		}
	}
	return nil
}

func printJSON(w io.Writer, projects []catalog.Project) error {
	rows := make([]projectJSON, len(projects))
	for i, p := range projects {
		rows[i] = projectJSON{p.ID, p.Name, p.Path, p.Description, p.Editor, p.AddedAt, p.LastAccessed}
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rows)
}
