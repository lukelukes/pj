package main

import "fmt"

type RmCmd struct {
	Name string `arg:"" help:"Project name or path to remove" completion:"pj list -n"`
}

func (cmd *RmCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	if err := g.Cat.Remove(project.ID); err != nil {
		return fmt.Errorf("failed to remove project %q: %w", project.Name, err)
	}

	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Removed: %s\n", project.Name)
	return nil
}
