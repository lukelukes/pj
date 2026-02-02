package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"pj/internal/catalog"
	"pj/internal/ui"
	"strings"

	"github.com/charmbracelet/huh"
)

type CreateCmd struct{}

type createResult struct {
	Name        string
	Location    string
	Description string
	Editor      string
	Git         bool
}

func validateCreateName(name string) error {
	err := catalog.ValidateName(name)
	if errors.Is(err, catalog.ErrEmptyName) {
		return errors.New("Name cannot be empty")
	}
	return err
}

func (cmd *CreateCmd) Run(g *Globals) error {
	var name string
	var description string
	var editor string
	gitInit := true

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	location := cwd

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Value(&name).
				Validate(validateCreateName),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Location").
				Description("Press Enter to accept, or type a new path").
				Value(&location),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Description (optional)").
				Placeholder("Press Enter to skip").
				Value(&description),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Editor (optional)").
				Placeholder("Press Enter to skip").
				Value(&editor),
		),
		huh.NewGroup(
			huh.NewSelect[bool]().
				Title("Initialize git repository?").
				Options(
					huh.NewOption("Yes (recommended)", true).Selected(true),
					huh.NewOption("No", false),
				).
				Value(&gitInit),
		),
	).WithTheme(ui.WizardTheme())

	if err := form.Run(); err != nil {
		return handleCreateFormError(err)
	}

	result := createResult{
		Name:        strings.TrimSpace(name),
		Location:    strings.TrimSpace(location),
		Description: strings.TrimSpace(description),
		Editor:      strings.TrimSpace(editor),
		Git:         gitInit,
	}

	projectPath, err := createProjectDir(result.Location, result.Name)
	if err != nil {
		return err
	}

	if result.Git {
		if err := initGitRepo(g, projectPath); err != nil {
			return err
		}
	}

	if err := registerProject(g, result, projectPath); err != nil {
		return err
	}

	renderCreateSummary(g, result)
	return nil
}

func createProjectDir(location, name string) (string, error) {
	projectPath := filepath.Join(location, name)
	if _, err := os.Stat(projectPath); err == nil {
		return "", fmt.Errorf("Directory already exists: %s", projectPath)
	}
	if err := os.Mkdir(projectPath, 0o755); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return "", fmt.Errorf("Permission denied: %s", projectPath)
		}
		return "", fmt.Errorf("creating directory: %w", err)
	}
	return projectPath, nil
}

func initGitRepo(g *Globals, projectPath string) error {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(g.Out, "⚠ Git not found, skipping initialization")
		return nil
	}
	cmd := exec.Command("git", "init", projectPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("initializing git repository: %w", err)
	}
	return createGitignore(projectPath)
}

func createGitignore(projectPath string) error {
	content := strings.Join([]string{
		".DS_Store",
		"Thumbs.db",
		"",
		".idea/",
		".vscode/",
		"*.swp",
		"",
		"/dist/",
		"/build/",
		"/out/",
		"",
		"/vendor/",
		"/node_modules/",
		"",
	}, "\n")
	return os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(content), 0o644)
}

func registerProject(g *Globals, result createResult, projectPath string) error {
	p := catalog.NewProject(result.Name, projectPath).
		WithDescription(result.Description).
		WithEditor(result.Editor)
	if err := g.Cat.Add(p); err != nil {
		return fmt.Errorf("adding project to catalog: %w", err)
	}
	if err := g.Cat.Save(); err != nil {
		return fmt.Errorf("saving catalog: %w", err)
	}
	return nil
}

func handleCreateFormError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	return err
}

func gitLabel(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func renderCreateSummary(g *Globals, r createResult) {
	projectPath := filepath.Join(r.Location, r.Name)
	checks := []string{"Directory created"}
	if r.Git {
		checks = append(checks, "Git initialized")
	}
	checks = append(checks, "Added to catalog")
	output := ui.RenderSuccess(r.Name, projectPath, checks)
	fmt.Fprint(g.Out, output)
}
