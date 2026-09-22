package main

import (
	"encoding/json"
	"fmt"
	"io"
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
	OutputTags  Output = "tags"
)

type projectJSON struct {
	Tags         []string  `json:"tags"`
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Description  string    `json:"description"`
	Editor       string    `json:"editor"`
	AddedAt      time.Time `json:"added_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

func printProjects(w io.Writer, projects []catalog.Project, output Output) error {
	projects = slices.Clone(projects)
	sortProjects(projects)
	switch output {
	case OutputNames, OutputPaths:
		return printField(w, projects, output)
	case OutputTags:
		return printLines(w, catalog.TagSet(projects))
	case OutputJSON:
		rows := make([]projectJSON, len(projects))
		for i, p := range projects {
			rows[i] = projectJSON{append([]string{}, p.Tags...), p.ID, p.Name, p.Path, p.Description, p.Editor, p.AddedAt, p.LastAccessed}
		}
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(rows)
	default:
		return fmt.Errorf("unknown output %q (valid outputs: table, names, paths, json, tags)", output)
	}
}

func printField(w io.Writer, projects []catalog.Project, output Output) error {
	values := make([]string, len(projects))
	for i, p := range projects {
		values[i] = p.Name
		if output == OutputPaths {
			values[i] = p.Path
		}
	}
	return printLines(w, values)
}

func printLines(w io.Writer, values []string) error {
	for _, value := range values {
		if _, err := fmt.Fprintln(w, value); err != nil {
			return err
		}
	}
	return nil
}
