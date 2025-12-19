package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"pj/internal/catalog"
	"pj/internal/config"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"
)

type Globals struct {
	Cat catalog.Catalog
	Out io.Writer
}

type CLI struct {
	Add    AddCmd    `cmd:"" help:"Add a project to the catalog"`
	List   ListCmd   `cmd:"" help:"List projects in the catalog"`
	Rm     RmCmd     `cmd:"" help:"Remove a project from the catalog"`
	Open   OpenCmd   `cmd:"" help:"Open a project (cd to directory)"`
	Edit   EditCmd   `cmd:"" help:"Edit project metadata"`
	Search SearchCmd `cmd:"" help:"Search for projects"`

	CatalogPath string `name:"catalog" short:"c" help:"Path to catalog file"`
}

func (c *CLI) AfterApply(ctx *kong.Context) error {
	// Determine catalog path
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

	globals := &Globals{Cat: cat, Out: os.Stdout}
	ctx.Bind(globals)
	return nil
}

type AddCmd struct {
	Path string   `arg:"" help:"Path to the project directory"`
	Name string   `short:"n" help:"Project name (defaults to directory name)"`
	Tags []string `short:"t" help:"Tags to add to the project"`
}

func (cmd *AddCmd) Run(g *Globals) error {
	cat := g.Cat
	path, err := filepath.Abs(cmd.Path)
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
	if len(cmd.Tags) > 0 {
		p = p.WithTags(cmd.Tags...)
	}

	p = p.WithTypes(catalog.DetectProjectTypes(path)...)

	if err := cat.Add(p); err != nil {
		return fmt.Errorf("failed to add project %q: %w", name, err)
	}

	if err := cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Added: %s (%s)\n", p.Name, p.Path)
	return nil
}

type ListCmd struct {
	Status string   `short:"s" help:"Filter by status (active, archived, abandoned)"`
	Types  []string `short:"T" help:"Filter by project types (matches any)"`
	Tags   []string `short:"t" help:"Filter by tags (all must match)"`
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

type RmCmd struct {
	Name string `arg:"" help:"Project name or path to remove"`
}

func (cmd *RmCmd) Run(g *Globals) error {
	cat := g.Cat
	projects := cat.Search(cmd.Name)

	if len(projects) == 0 {
		return fmt.Errorf("no project found matching: %s", cmd.Name)
	}

	if len(projects) > 1 {
		fmt.Fprintln(g.Out, "Multiple projects match. Please be more specific:")
		for _, p := range projects {
			fmt.Fprintf(g.Out, "  - %s (%s)\n", p.Name, p.Path)
		}
		return nil
	}

	p := projects[0]
	if err := cat.Remove(p.ID); err != nil {
		return fmt.Errorf("failed to remove project %q: %w", p.Name, err)
	}

	if err := cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Removed: %s\n", p.Name)
	return nil
}

type OpenCmd struct {
	Name string `arg:"" help:"Project name to open"`
}

func (cmd *OpenCmd) Run(g *Globals) error {
	cat := g.Cat
	projects := cat.Search(cmd.Name)

	if len(projects) == 0 {
		return fmt.Errorf("no project found matching: %s", cmd.Name)
	}

	if len(projects) > 1 {
		fmt.Fprintln(g.Out, "Multiple projects match. Please be more specific:")
		for _, p := range projects {
			fmt.Fprintf(g.Out, "  - %s (%s)\n", p.Name, p.Path)
		}
		return nil
	}

	p := projects[0]
	p.Touch()
	if err := cat.Update(p); err != nil {
		return fmt.Errorf("failed to update project %q: %w", p.Name, err)
	}
	if err := cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintln(g.Out, p.Path)
	return nil
}

type EditCmd struct {
	Name   string   `arg:"" help:"Project name to edit"`
	Status string   `short:"s" help:"Set status (active, archived, abandoned)"`
	AddTag []string `help:"Add tags"`
	RmTag  []string `help:"Remove tags"`
	Notes  string   `help:"Set notes"`
}

func (cmd *EditCmd) Run(g *Globals) error {
	cat := g.Cat
	projects := cat.Search(cmd.Name)

	if len(projects) == 0 {
		return fmt.Errorf("no project found matching: %s", cmd.Name)
	}

	if len(projects) > 1 {
		fmt.Fprintln(g.Out, "Multiple projects match. Please be more specific:")
		for _, p := range projects {
			fmt.Fprintf(g.Out, "  - %s (%s)\n", p.Name, p.Path)
		}
		return nil
	}

	p := projects[0]
	if cmd.Status != "" {
		p.Status = catalog.Status(cmd.Status)
	}
	for _, tag := range cmd.AddTag {
		p.AddTag(tag)
	}
	for _, tag := range cmd.RmTag {
		p.RemoveTag(tag)
	}
	if cmd.Notes != "" {
		p.Notes = cmd.Notes
	}

	if err := cat.Update(p); err != nil {
		return fmt.Errorf("failed to update project %q: %w", p.Name, err)
	}

	if err := cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Updated: %s\n", p.Name)
	return nil
}

type SearchCmd struct {
	Query string `arg:"" help:"Search query"`
}

func (cmd *SearchCmd) Run(g *Globals) error {
	cat := g.Cat
	projects := cat.Search(cmd.Query)

	if len(projects) == 0 {
		fmt.Fprintln(g.Out, "No projects found.")
		return nil
	}

	for _, p := range projects {
		fmt.Fprintf(g.Out, "%s  %s\n", p.Name, p.Path)
	}

	return nil
}

func shortenPath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
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
