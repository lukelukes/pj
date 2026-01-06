package main

import (
	"fmt"
	"pj/internal/catalog"
)

type EditCmd struct {
	Name   string `arg:"" help:"Project name to edit" completion:"pj list -n"`
	Editor string `help:"Set editor command (e.g., code, nvim)"`
}

func (cmd *EditCmd) applyEdits(p *catalog.Project) {
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
