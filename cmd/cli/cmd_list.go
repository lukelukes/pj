package main

import (
	"fmt"
	"pj/internal/catalog"
	"strings"
	"text/tabwriter"
)

type ListCmd struct {
	Names bool `short:"n" help:"Output only project names (one per line)"`
}

func (cmd *ListCmd) Run(g *Globals) error {
	projects := g.Cat.List()

	if cmd.Names {
		for _, p := range projects {
			fmt.Fprintln(g.Out, p.Name)
		}
		return nil
	}

	if len(projects) == 0 {
		fmt.Fprintln(g.Out, "No projects found.")
		return nil
	}

	return cmd.printProjects(g, projects)
}

func (cmd *ListCmd) printProjects(g *Globals, projects []catalog.Project) error {
	w := tabwriter.NewWriter(g.Out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tTYPES\tSTATUS\tTAGS")
	fmt.Fprintln(w, "----\t----\t-----\t------\t----")

	for _, p := range projects {
		tags := strings.Join(p.Tags, ", ")
		types := make([]string, len(p.Types))
		for i, t := range p.Types {
			types[i] = string(t)
		}
		typesStr := strings.Join(types, ", ")
		if typesStr == "" {
			typesStr = "unknown"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			p.Name, shortenPath(p.Path), typesStr, p.Status, tags)
	}

	return w.Flush()
}
