package main

import (
	"errors"
	"fmt"
	"os"
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
	renderCreateSummary(g, result)
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
	fields := []ui.Field{
		{Label: "Name", Value: r.Name},
		{Label: "Location", Value: r.Location},
		{Label: "Description", Value: r.Description, Optional: true},
		{Label: "Editor", Value: r.Editor, Optional: true},
		{Label: "Git", Value: gitLabel(r.Git)},
	}
	output := ui.RenderWizard("Create new project", fields, -1)
	fmt.Fprint(g.Out, output)
}
