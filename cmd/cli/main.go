package main

import (
	"fmt"
	"os"
	"pj/cmd/cli/render"
	"pj/internal/catalog"
	"pj/internal/config"

	"github.com/alecthomas/kong"
)

type CLI struct {
	Add        AddCmd        `cmd:"" aliases:"a" help:"Add a project to the catalog"`
	List       ListCmd       `cmd:"" aliases:"ls" help:"List projects in the catalog"`
	Rm         RmCmd         `cmd:"" help:"Remove a project from the catalog"`
	Open       OpenCmd       `cmd:"" aliases:"o" help:"Open project in editor"`
	Edit       EditCmd       `cmd:"" aliases:"e" help:"Edit project metadata"`
	Show       ShowCmd       `cmd:"" help:"Show project details"`
	Cd         CdCmd         `cmd:"" help:"Change directory to project (requires shell integration)"`
	Init       InitCmd       `cmd:"" help:"Generate shell integration"`
	Completion CompletionCmd `cmd:"" help:"Generate shell completions"`

	CatalogPath string `name:"catalog" short:"c" help:"Path to catalog file"`
}

func (c *CLI) AfterApply(ctx *kong.Context) error {
	catalogPath := c.CatalogPath
	if catalogPath == "" {
		catalogPath = config.DefaultCatalogPath()
	}

	cat, err := catalog.NewYAMLCatalog(catalogPath)
	if err != nil {
		return fmt.Errorf("failed to create catalog: %w", err)
	}
	if err := cat.Load(); err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	globals := &Globals{
		Cat:    cat,
		Out:    os.Stdout,
		Render: render.NewLipglossRendererAuto(os.Stdout),
	}
	ctx.Bind(globals)
	return nil
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("pj"),
		kong.Description("Project tracker and launcher"),
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
