package main

import (
	"fmt"
	"pj/internal/catalog"
	"slices"
)

type RmCmd struct {
	Projects []string `arg:"" name:"project" help:"Project names or paths to remove. Nothing is removed unless every one resolves." completion:"pj list -n"`
}

func (cmd *RmCmd) Run(g *Globals) error {
	var targets []catalog.Project
	for _, query := range cmd.Projects {
		project, err := findProject(g.Cat, query)
		if err != nil {
			if handleFindError(g.Out, err) {
				return nil
			}
			return err
		}
		if !slices.ContainsFunc(targets, func(p catalog.Project) bool { return p.ID == project.ID }) {
			targets = append(targets, project)
		}
	}

	for _, project := range targets {
		if err := g.Cat.Remove(project.ID); err != nil {
			return fmt.Errorf("failed to remove project %q: %w", project.Name, err)
		}
	}

	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	for _, project := range targets {
		fmt.Fprintf(g.Out, "Removed: %s\n", project.Name)
	}
	return nil
}
