package main

import (
	"fmt"
	"os"
	"pj/cmd/cli/render"
	"slices"
	"time"
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

	items := make([]render.ProjectListItem, len(projects))
	for i, p := range projects {
		items[i] = render.ProjectListItem{
			Name:        p.Name,
			Path:        p.Path,
			Description: p.Description,
			Timestamp:   getMtime(p.Path),
		}
	}
	slices.SortFunc(items, func(a, b render.ProjectListItem) int {
		return b.Timestamp.Compare(a.Timestamp)
	})

	view := render.ProjectListView{Items: items}
	output := g.Render.RenderProjectList(view)
	_, err := fmt.Fprint(g.Out, output)
	return err
}

func getMtime(path string) time.Time {
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}
