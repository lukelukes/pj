package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"pj/internal/catalog"
	"pj/internal/config"
	"strings"
	"text/tabwriter"

	"github.com/alecthomas/kong"
)

type Globals struct {
	Cat    catalog.Catalog
	Out    io.Writer
	RunCmd func(name string, args ...string) error
}

func defaultRunCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type AmbiguousMatchError struct {
	Query   string
	Matches []catalog.Project
}

func (e *AmbiguousMatchError) Error() string {
	return fmt.Sprintf("multiple projects match %q", e.Query)
}

func (e *AmbiguousMatchError) WriteMatches(w io.Writer) {
	fmt.Fprintln(w, "Multiple projects match. Please be more specific:")
	for _, p := range e.Matches {
		fmt.Fprintf(w, "  - %s (%s)\n", p.Name, p.Path)
	}
}

func handleFindError(w io.Writer, err error) bool {
	var ambErr *AmbiguousMatchError
	if errors.As(err, &ambErr) {
		ambErr.WriteMatches(w)
		return true
	}
	return false
}

func findProject(cat catalog.Catalog, query string) (catalog.Project, error) {
	projects := cat.Search(query)
	if len(projects) == 0 {
		return catalog.Project{}, fmt.Errorf("no project found matching: %s", query)
	}
	if len(projects) > 1 {
		return catalog.Project{}, &AmbiguousMatchError{Query: query, Matches: projects}
	}
	return projects[0], nil
}

func resolveEditor(project catalog.Project) (string, error) {
	editor := project.Editor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vim"
	}
	if _, err := exec.LookPath(editor); err != nil {
		return "", fmt.Errorf("editor %q not found in PATH", editor)
	}
	return editor, nil
}

type CLI struct {
	Add    AddCmd    `cmd:"" aliases:"a" help:"Add a project to the catalog"`
	List   ListCmd   `cmd:"" aliases:"ls" help:"List projects in the catalog"`
	Rm     RmCmd     `cmd:"" help:"Remove a project from the catalog"`
	Open   OpenCmd   `cmd:"" aliases:"o" help:"Open project in editor"`
	Edit   EditCmd   `cmd:"" aliases:"e" help:"Edit project metadata"`
	Search SearchCmd `cmd:"" aliases:"s" help:"Search for projects"`
	Show   ShowCmd   `cmd:"" help:"Show project details"`
	Init   InitCmd   `cmd:"" help:"Generate shell integration"`

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
	path, err := config.ExpandPath(cmd.Path)
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

type RmCmd struct {
	Name string `arg:"" help:"Project name or path to remove"`
}

func (cmd *RmCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	if err := g.Cat.Remove(project.ID); err != nil {
		return fmt.Errorf("failed to remove project %q: %w", project.Name, err)
	}

	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Removed: %s\n", project.Name)
	return nil
}

type OpenCmd struct {
	Name string `arg:"" help:"Project name or partial match"`
}

func (cmd *OpenCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	if _, err := os.Stat(project.Path); os.IsNotExist(err) {
		return fmt.Errorf("project path no longer exists: %s\nRun 'pj rm %s' to remove from catalog",
			project.Path, project.Name)
	}

	editor, err := resolveEditor(project)
	if err != nil {
		return err
	}

	project.Touch()
	if err := g.Cat.Update(project); err != nil {
		return fmt.Errorf("failed to update project %q: %w", project.Name, err)
	}
	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	runCmd := g.RunCmd
	if runCmd == nil {
		runCmd = defaultRunCmd
	}
	return runCmd(editor, project.Path)
}

type EditCmd struct {
	Name   string   `arg:"" help:"Project name to edit"`
	Status string   `short:"s" help:"Set status (active, archived, abandoned)"`
	AddTag []string `help:"Add tags"`
	RmTag  []string `help:"Remove tags"`
	Notes  string   `help:"Set notes"`
	Editor string   `help:"Set editor command (e.g., code, nvim)"`
}

func (cmd *EditCmd) applyEdits(p *catalog.Project) {
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
	if cmd.Editor != "" {
		p.Editor = cmd.Editor
	}
}

func (cmd *EditCmd) Run(g *Globals) error {
	project, err := findProject(g.Cat, cmd.Name)
	if err != nil {
		if handleFindError(g.Out, err) {
			return nil
		}
		return err
	}

	cmd.applyEdits(&project)

	if err := g.Cat.Update(project); err != nil {
		return fmt.Errorf("failed to update project %q: %w", project.Name, err)
	}

	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("failed to save catalog: %w", err)
	}

	fmt.Fprintf(g.Out, "Updated: %s\n", project.Name)
	return nil
}

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

type ShowCmd struct {
	Name string `arg:"" help:"Project name"`
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

type InitCmd struct{}

func (cmd *InitCmd) Run(g *Globals) error { //nolint:unparam // error required by kong interface
	fmt.Fprint(g.Out, shellScript)
	return nil
}

const shellScript = `# pj shell integration
# Add to ~/.bashrc or ~/.zshrc: eval "$(pj init)"

pj() {
    case "$1" in
        cd)
            if [ -z "$2" ]; then
                echo "Usage: pj cd <project>" >&2
                return 1
            fi
            # Capture both stdout and exit code - propagate real errors
            dir="$(command pj show "$2" --path 2>&1)"
            if [ $? -ne 0 ]; then
                echo "pj: $dir" >&2
                return 1
            fi
            if [ ! -d "$dir" ]; then
                echo "pj: path no longer exists: $dir" >&2
                return 1
            fi
            builtin cd -- "$dir"
            ;;
        *)
            command pj "$@"
            ;;
    esac
}
`

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
