package main

import "fmt"

type SearchCmd struct {
	Query string `arg:"" help:"Search query"`
}

func (cmd *SearchCmd) Run(g *Globals) error { //nolint:unparam // error required by kong interface
	cat := g.Cat
	projects := cat.Search(cmd.Query)

	if len(projects) == 0 {
		fmt.Fprintln(g.Out, "No projects found.")
	} else {
		for _, p := range projects {
			fmt.Fprintf(g.Out, "%s  %s\n", p.Name, p.Path)
		}
	}

	return nil
}
