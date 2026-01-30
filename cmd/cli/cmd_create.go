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

func validateCreateName(name string) error {
	err := catalog.ValidateName(name)
	if errors.Is(err, catalog.ErrEmptyName) {
		return errors.New("Name cannot be empty")
	}
	return err
}

func (cmd *CreateCmd) Run(g *Globals) error {
	var name string

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
	).WithTheme(ui.WizardTheme())

	if err := form.Run(); err != nil {
		return handleCreateFormError(err)
	}

	renderCreateSummary(g, strings.TrimSpace(name), strings.TrimSpace(location))
	return nil
}

func handleCreateFormError(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return nil
	}
	return err
}

func renderCreateSummary(g *Globals, name, location string) {
	fields := []ui.Field{
		{Label: "Name", Value: name},
		{Label: "Location", Value: location},
	}
	output := ui.RenderWizard("Create new project", fields, -1)
	fmt.Fprint(g.Out, output)
}
