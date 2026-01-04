package main

import (
	"fmt"
	"os"
)

type OpenCmd struct {
	Name string `arg:"" help:"Project name or partial match" completion:"pj list -n"`
}

func (cmd *OpenCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	if _, err := os.Stat(project.Path); os.IsNotExist(err) {
		return fmt.Errorf("project path no longer exists: %s\nRun 'pj rm %s' to remove from catalog",
			project.Path, project.Name)
	}

	editor, err := resolveEditor(project)
	if err != nil {
		return err
	}

	project.Touch()
	if err := g.Cat.Update(project); err != nil {
		return fmt.Errorf("failed to update project %q: %w", project.Name, err)
	}
	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	runCmd := g.RunCmd
	if runCmd == nil {
		runCmd = defaultRunCmd
	}
	return runCmd(editor, project.Path)
}
