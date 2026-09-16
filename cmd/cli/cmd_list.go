package main

import (
	"fmt"
	"os"
	"pj/cmd/cli/render"
	"slices"
	"time"
)

type ListCmd struct {
	Selector `embed:""`
	Output   Output `short:"o" enum:"table,names,paths,json,tags" default:"table" help:"Output format: table, names, paths, json, tags"`
	Names    bool   `short:"n" hidden:"" help:"Alias for --output names"`
}

func (cmd *ListCmd) Run(g *Globals) error {
	projects, err := cmd.Select(g.Cat)
	if err != nil {
		return err
	}
	output := cmd.Output
	if cmd.Names {
		output = OutputNames
	}
	if output != "" && output != OutputTable {
		return printProjects(g.Out, projects, output)
	}

	items := make([]render.ProjectListItem, len(projects))
	for i, p := range projects {
		items[i] = render.ProjectListItem{
			Name:        p.Name,
			Path:        p.Path,
			Description: p.Description,
			Tags:        p.Tags,
			Timestamp:   getMtime(p.Path),
		}
	}
	slices.SortFunc(items, func(a, b render.ProjectListItem) int {
		return b.Timestamp.Compare(a.Timestamp)
	})

	view := render.ProjectListView{Items: items}
	rendered := g.Render.RenderProjectList(view)
	_, err = fmt.Fprint(g.Out, rendered)
	return err
}

func getMtime(path string) time.Time {
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}
