package main

import (
	"errors"
	"fmt"
	"pj/internal/catalog"
)

type EditCmd struct {
	Tags      []string `name:"tag" short:"t" help:"Add tags"`
	Untag     []string `help:"Remove tags" xor:"remove-tags"`
	ClearTags bool     `help:"Remove all tags before adding new ones" xor:"remove-tags"`
	Desc      string   `help:"Set project description"`
	Name      string   `arg:"" help:"Project name to edit" completion:"pj list -n"`
	Editor    string   `help:"Set editor command (e.g., code, nvim)"`
}

func (cmd *EditCmd) applyEdits(p *catalog.Project) error {
	if cmd.ClearTags && len(cmd.Untag) > 0 {
		return errors.New("--clear-tags and --untag cannot be combined")
	}
	tags, err := catalog.NormalizeTags(cmd.Tags)
	if err != nil {
		return err
	}
	untag, err := catalog.NormalizeTags(cmd.Untag)
	if err != nil {
		return err
	}
	edited := p.WithTags(p.Tags)
	if cmd.ClearTags {
		edited.Tags = nil
	}
	edited.RemoveTags(untag...)
	if err := edited.AddTags(tags...); err != nil {
		return err
	}
	*p = edited
	if cmd.Desc != "" {
		p.Description = cmd.Desc
	}
	if cmd.Editor != "" {
		p.Editor = cmd.Editor
	}
	return nil
}

func (cmd *EditCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	if err := cmd.applyEdits(&project); err != nil {
		return err
	}

	if err := g.Cat.Update(project); err != nil {
		return fmt.Errorf("failed to update project %q: %w", project.Name, err)
	}

	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Updated: %s\n", project.Name)
	return nil
}
