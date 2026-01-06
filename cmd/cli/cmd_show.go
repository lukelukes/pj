package main

import "fmt"

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
	if project.Editor != "" {
		fmt.Fprintf(g.Out, "Editor: %s\n", project.Editor)
	}
	return nil
}
