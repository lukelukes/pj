package main

import (
	"fmt"
	"pj/internal/catalog"
)

type EditCmd struct {
	Name   string   `arg:"" help:"Project name to edit"`
	Status string   `short:"s" help:"Set status (active, archived, abandoned)"`
	AddTag []string `help:"Add tags"`
	RmTag  []string `help:"Remove tags"`
	Notes  string   `help:"Set notes"`
	Editor string   `help:"Set editor command (e.g., code, nvim)"`
}

func (cmd *EditCmd) applyEdits(p *catalog.Project) {
	if cmd.Status != "" {
		p.Status = catalog.Status(cmd.Status)
	}
	for _, tag := range cmd.AddTag {
		p.AddTag(tag)
	}
	for _, tag := range cmd.RmTag {
		p.RemoveTag(tag)
	}
	if cmd.Notes != "" {
		p.Notes = cmd.Notes
	}
	if cmd.Editor != "" {
		p.Editor = cmd.Editor
	}
}

func (cmd *EditCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	cmd.applyEdits(&project)

	if err := g.Cat.Update(project); err != nil {
		return fmt.Errorf("failed to update project %q: %w", project.Name, err)
	}

	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Updated: %s\n", project.Name)
	return nil
}
