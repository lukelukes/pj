package main

import (
	"fmt"
	"io"
	"os"
	"pj/internal/catalog"
	"pj/internal/detect"
	"slices"
	"strings"
)

type TagCmd struct {
	Detect TagDetectCmd `cmd:"" help:"Detect language tags from project files"`
}

type TagDetectCmd struct {
	Selector `embed:""`
	Apply    bool `help:"Save detected tags to the catalog"`
}

type tagChange struct {
	project catalog.Project
	added   []string
}

func (cmd *TagDetectCmd) Run(g *Globals) error {
	targets, err := cmd.targets(g.Cat)
	if err != nil {
		return err
	}
	changes := planTags(targets)
	if cmd.Apply {
		if err := applyTags(g.Cat, changes); err != nil {
			return err
		}
	}
	return reportTags(g.Out, changes, cmd.Apply)
}

func (cmd *TagDetectCmd) targets(cat catalog.Catalog) ([]catalog.Project, error) {
	projects, err := cmd.Select(cat)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(projects, func(p catalog.Project) bool {
		info, err := os.Stat(p.Path)
		return err != nil || !info.IsDir()
	}), nil
}

func planTags(projects []catalog.Project) []tagChange {
	var changes []tagChange
	for _, p := range projects {
		added := slices.DeleteFunc(detect.Detect(os.DirFS(p.Path)), p.HasTag)
		if len(added) > 0 {
			changes = append(changes, tagChange{p, added})
		}
	}
	return changes
}

func applyTags(cat catalog.Catalog, changes []tagChange) error {
	for _, change := range changes {
		p := change.project
		if err := p.AddTags(change.added...); err != nil {
			return err
		}
		if err := cat.Update(p); err != nil {
			return fmt.Errorf("tagging %q: %w", p.Name, err)
		}
	}
	if len(changes) == 0 {
		return nil
	}
	if err := cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}
	return nil
}

func reportTags(w io.Writer, changes []tagChange, applied bool) error {
	for _, change := range changes {
		if _, err := fmt.Fprintf(w, "  %s  +%s\n", change.project.Name, strings.Join(change.added, " +")); err != nil {
			return err
		}
	}
	var footer string
	switch {
	case len(changes) == 0:
		footer = "Nothing to do."
	case applied:
		footer = fmt.Sprintf("Tagged %d projects.", len(changes))
	default:
		footer = fmt.Sprintf("%d projects would change. Re-run with --apply to write.", len(changes))
	}
	_, err := fmt.Fprintln(w, footer)
	return err
}
