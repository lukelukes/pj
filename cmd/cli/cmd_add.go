package main

import (
	"fmt"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"pj/internal/config"
)

type AddCmd struct {
	Path string `arg:"" help:"Path to the project directory"`
	Name string `short:"n" help:"Project name (defaults to directory name)"`
}

func (cmd *AddCmd) Run(g *Globals) error {
	cat := g.Cat
	path, err := config.ExpandPath(cmd.Path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	name := cmd.Name
	if name == "" {
		name = filepath.Base(path)
	}

	p := catalog.NewProject(name, path)

	if err := cat.Add(p); err != nil {
		return fmt.Errorf("failed to add project %q: %w", name, err)
	}

	if err := cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Added: %s (%s)\n", p.Name, p.Path)
	return nil
}
