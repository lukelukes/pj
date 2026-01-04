package main

import (
	"fmt"
	"pj/internal/catalog"
	"strings"
	"text/tabwriter"
)

type ListCmd struct {
	Status string   `short:"s" help:"Filter by status (active, archived, abandoned)"`
	Types  []string `short:"T" help:"Filter by project types (matches any)"`
	Tags   []string `short:"t" help:"Filter by tags (all must match)"`
	Recent bool     `short:"r" help:"Sort by last accessed (newest first)"`
	JSON   bool     `help:"Output as JSON"`
}

func (cmd *ListCmd) Run(g *Globals) error {
	cat := g.Cat
	opts := catalog.FilterOptions{}

	if cmd.Status != "" {
		opts.Status = catalog.Status(cmd.Status)
	}
	if len(cmd.Types) > 0 {
		for _, t := range cmd.Types {
			opts.Types = append(opts.Types, catalog.ProjectType(t))
		}
	}
	if len(cmd.Tags) > 0 {
		opts.Tags = cmd.Tags
	}
	if cmd.Recent {
		opts.SortBy = catalog.SortByLastAccessed
		opts.Descending = true
	}

	projects := cat.Filter(opts)

	if len(projects) == 0 {
		fmt.Fprintln(g.Out, "No projects found.")
		return nil
	}

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
