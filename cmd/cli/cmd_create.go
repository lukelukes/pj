package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"pj/internal/catalog"
	"pj/internal/config"
	"pj/internal/create"
	"pj/internal/ui"
	"pj/internal/ui/wizardtea"
	"strings"

	"github.com/charmbracelet/x/term"
)

type CreateCmd struct {
	Tags    []string `name:"tag" short:"t" help:"Add tags (comma-separated or repeated)"`
	Name    string   `arg:"" optional:"" help:"Project name"`
	At      string   `help:"Parent directory (defaults to the working directory)"`
	Desc    string   `help:"Short description"`
	Editor  string   `help:"Editor command for this project"`
	NoGit   bool     `help:"Skip git initialization"`
	Adopt   bool     `help:"Adopt the directory if it already exists"`
	NoInput bool     `help:"Never prompt; fail when information is missing"`
}

func (cmd *CreateCmd) Run(g *Globals) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	location, err := resolveLocation(cmd.At, cwd)
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	env := create.CatalogEnv{Cat: g.Cat}

	req := create.Request{
		Name:        cmd.Name,
		Tags:        cmd.Tags,
		Location:    location,
		Description: cmd.Desc,
		Editor:      cmd.Editor,
		Git:         !cmd.NoGit,
	}

	adopt := cmd.Adopt
	if cmd.needsWizard() {
		outcome, err := cmd.runWizard(env, req, home)
		if err != nil {
			return err
		}
		if outcome.Cancelled {
			return nil
		}
		req = toRequest(outcome.Draft)
		adopt = outcome.Adopt
	}

	plan := create.BuildPlan(req, env)
	if plan.Blocker != nil {
		return plan.Blocker
	}
	if plan.PathTaken != nil {
		return fmt.Errorf("%s is already tracked as %q", ui.DisplayPath(plan.Path, home), plan.PathTaken.Name)
	}
	if plan.NeedsAdopt() && !adopt {
		return fmt.Errorf("%s already exists; pass --adopt to add it to the catalog", ui.DisplayPath(plan.Path, home))
	}

	return runPlan(g, plan, adopt, home)
}

func resolveLocation(at, cwd string) (string, error) {
	if at == "" {
		return cwd, nil
	}
	expanded, err := config.ExpandPath(at)
	if err != nil {
		return "", fmt.Errorf("invalid --at path: %w", err)
	}
	return expanded, nil
}

func (cmd *CreateCmd) needsWizard() bool {
	if cmd.NoInput || cmd.Name != "" {
		return false
	}
	return term.IsTerminal(os.Stdin.Fd())
}

func (cmd *CreateCmd) runWizard(env create.Env, req create.Request, home string) (ui.Outcome, error) {
	session := ui.Session{
		Draft:      toDraft(req),
		Preview:    previewFunc(env, home),
		EditorHint: os.Getenv("EDITOR"),
		Home:       home,
	}

	return wizardtea.Runner{Palette: ui.DefaultPalette()}.Run(session)
}

func runPlan(g *Globals, plan create.Plan, adopt bool, home string) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)

	createdDir := !plan.DirExists
	go func() {
		if _, ok := <-sigCh; ok {
			if createdDir {
				os.RemoveAll(plan.Path)
			}
			os.Exit(130)
		}
	}()

	svc := create.Service{Cat: g.Cat, Run: quietRun, GitFound: gitAvailable}
	outcome, err := svc.Execute(plan, adopt)
	if err != nil {
		return err
	}

	jumped := writeCdFile(outcome.Path)
	fmt.Fprint(g.Out, ui.RenderReceipt(ui.Receipt{
		Name:    plan.Name,
		Path:    ui.DisplayPath(outcome.Path, home),
		Steps:   outcome.Steps,
		Adopted: outcome.Adopted,
		Jumped:  jumped,
	}, ui.DefaultPalette()))

	if !jumped {
		fmt.Fprintf(g.Out, "\nRun: cd %s\n", outcome.Path)
	}
	return nil
}

func previewFunc(env create.Env, home string) ui.PreviewFunc {
	return func(d ui.Draft) ui.Preview {
		return toPreview(create.BuildPlan(toRequest(d), env), home)
	}
}

func toPreview(p create.Plan, home string) ui.Preview {
	switch {
	case p.Blocker != nil:
		return ui.Preview{Severity: ui.SeverityBlock, Message: p.Blocker.Error()}
	case p.PathTaken != nil:
		return ui.Preview{
			Severity: ui.SeverityBlock,
			Message:  fmt.Sprintf("%s is already tracked as %q", ui.DisplayPath(p.Path, home), p.PathTaken.Name),
		}
	case p.DirExists:
		return ui.Preview{
			Path:       ui.DisplayPath(p.Path, home),
			Severity:   ui.SeverityNotice,
			NeedsAdopt: true,
			Message:    fmt.Sprintf("%s exists · %s", ui.DisplayPath(p.Path, home), pluralFiles(p.DirEntries)),
			AdoptHint:  "↵ adopt into catalog",
		}
	}

	preview := ui.Preview{Path: ui.DisplayPath(p.Path, home)}
	if p.WillInitGit() {
		preview.Facts = append(preview.Facts, "git init")
	}
	if nested := p.NestedIn(); nested != "" {
		preview.Facts = append(preview.Facts, "inside "+ui.DisplayPath(nested, home))
	}
	if len(p.Tags) > 0 {
		preview.Facts = append(preview.Facts, ui.FormatTags(p.Tags))
	}
	if p.Editor != "" {
		preview.Facts = append(preview.Facts, p.Editor)
	}
	if p.NameTaken != nil {
		preview.Severity = ui.SeverityNotice
		preview.Message = "name already used by " + ui.DisplayPath(p.NameTaken.Path, home)
	}
	return preview
}

func pluralFiles(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

func toRequest(d ui.Draft) create.Request {
	return create.Request{
		Name:        d.Name,
		Tags:        catalog.SplitTags(d.Tags),
		Location:    d.Location,
		Description: d.Description,
		Editor:      d.Editor,
		Git:         d.Git,
	}
}

func toDraft(r create.Request) ui.Draft {
	return ui.Draft{
		Name:        r.Name,
		Tags:        strings.Join(r.Tags, ", "),
		Location:    r.Location,
		Description: r.Description,
		Editor:      r.Editor,
		Git:         r.Git,
	}
}

func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func quietRun(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func writeCdFile(path string) bool {
	cdFile := os.Getenv("__PJ_CD_FILE")
	if cdFile == "" {
		return false
	}
	return os.WriteFile(cdFile, []byte(path), 0o600) == nil
}
