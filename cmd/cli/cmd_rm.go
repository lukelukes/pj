package main

import "fmt"

type RmCmd struct {
	Projects []string `arg:"" name:"project" help:"Project names or paths to remove. Nothing is removed unless every one resolves." completion:"pj list -n"`
}

func (cmd *RmCmd) Run(g *Globals) error {
	targets, ok, err := resolveProjects(g, cmd.Projects)
	if !ok {
		return err
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
