package main

import (
	"fmt"
	"strings"
)

type ShowCmd struct {
	Name string `arg:"" help:"Project name" completion:"pj list -n"`
	Path bool   `help:"Output only the path (for scripting)"`
}

func (cmd *ShowCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	if cmd.Path {
		fmt.Fprintln(g.Out, project.Path)
		return nil
	}

	fmt.Fprintf(g.Out, "Name:   %s\n", project.Name)
	fmt.Fprintf(g.Out, "Path:   %s\n", project.Path)
	fmt.Fprintf(g.Out, "Status: %s\n", project.Status)
	if project.Editor != "" {
		fmt.Fprintf(g.Out, "Editor: %s\n", project.Editor)
	}
	if len(project.Tags) > 0 {
		fmt.Fprintf(g.Out, "Tags:   %s\n", strings.Join(project.Tags, ", "))
	}
	if len(project.Types) > 0 {
		types := make([]string, len(project.Types))
		for i, t := range project.Types {
			types[i] = string(t)
		}
		fmt.Fprintf(g.Out, "Types:  %s\n", strings.Join(types, ", "))
	}
	if project.Notes != "" {
		fmt.Fprintf(g.Out, "Notes:  %s\n", project.Notes)
	}
	return nil
}
