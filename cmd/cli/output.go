package main

import (
	"cmp"
	"encoding/json"
	"errors"
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

func printProjects(w io.Writer, r render.Renderer, projects []catalog.Project, output Output, sort *catalog.Sort) error {
	order := defaultSort(output)
	if sort != nil {
		if output == OutputTags {
			return errors.New("sort: does not apply to -o tags, which lists each tag once in name order")
		}
		order = *sort
	}
	mtime := mtimes()
	projects = sortProjects(projects, order, mtime)
	switch output {
	case OutputTable:
		return printTable(w, r, projects, mtime)
	case OutputNames:
		return printLines(w, projects, func(p catalog.Project) string { return p.Name })
	case OutputPaths:
		return printLines(w, projects, func(p catalog.Project) string { return p.Path })
	case OutputTags:
		return printLines(w, catalog.TagSet(projects), func(tag string) string { return tag })
	case OutputJSON:
		return printJSON(w, projects)
	default:
		return fmt.Errorf("unknown output %q (valid outputs: table, names, paths, json, tags)", output)
	}
}

func defaultSort(output Output) catalog.Sort {
	if output == OutputTable {
		return catalog.Sort{Key: catalog.SortModified, Desc: true}
	}
	return catalog.Sort{Key: catalog.SortName}
}

func compareByName(a, b catalog.Project) int {
	return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Path, b.Path), cmp.Compare(a.ID, b.ID))
}

func sortProjects(projects []catalog.Project, order catalog.Sort, mtime func(string) time.Time) []catalog.Project {
	key := func(p catalog.Project) time.Time {
		switch order.Key {
		case catalog.SortOpened:
			return p.LastAccessed
		case catalog.SortAdded:
			return p.AddedAt
		case catalog.SortModified:
			return mtime(p.Path)
		default:
			return time.Time{}
		}
	}
	return slices.SortedStableFunc(slices.Values(projects), func(a, b catalog.Project) int {
		var c int
		if order.Key == catalog.SortName {
			c = compareByName(a, b)
		} else {
			c = key(a).Compare(key(b))
		}
		if order.Desc {
			c = -c
		}
		return cmp.Or(c, compareByName(a, b))
	})
}

func mtimes() func(string) time.Time {
	seen := map[string]time.Time{}
	return func(path string) time.Time {
		t, ok := seen[path]
		if !ok {
			t = getMtime(path)
			seen[path] = t
		}
		return t
	}
}

func printTable(w io.Writer, r render.Renderer, projects []catalog.Project, mtime func(string) time.Time) error {
	items := make([]render.ProjectListItem, len(projects))
	for i, p := range projects {
		items[i] = render.ProjectListItem{
			Name:        p.Name,
			Path:        p.Path,
			Description: p.Description,
			Timestamp:   mtime(p.Path),
			Tags:        p.Tags,
		}
	}
	_, err := fmt.Fprint(w, r.RenderProjectList(render.ProjectListView{Items: items}))
	return err
}

func getMtime(path string) time.Time {
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

func printLines[T any](w io.Writer, items []T, line func(T) string) error {
	for _, item := range items {
		if _, err := fmt.Fprintln(w, line(item)); err != nil {
			return err
		}
	}
	return nil
}

func printJSON(w io.Writer, projects []catalog.Project) error {
	rows := make([]projectJSON, len(projects))
	for i, p := range projects {
		rows[i] = projectJSON{append([]string{}, p.Tags...), p.ID, p.Name, p.Path, p.Description, p.Editor, p.AddedAt, p.LastAccessed}
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rows)
}
