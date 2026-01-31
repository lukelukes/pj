package main

import (
	"errors"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
)

func TestRenderCreateSummary(t *testing.T) {
	t.Run("shows collapsed name field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/home/user/projects"})

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "my-project")
	})

	t.Run("shows collapsed location field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/home/user/projects"})

		output := out.String()
		assert.Contains(t, output, "Location")
		assert.Contains(t, output, "/home/user/projects")
	})

	t.Run("shows both name and location fields", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev"})

		output := out.String()
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "my-project")
		assert.Contains(t, output, "Location")
		assert.Contains(t, output, "/tmp/dev")
	})

	t.Run("shows collapsed description field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Description: "A cool project"})

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Description")
		assert.Contains(t, output, "A cool project")
	})

	t.Run("omits empty description from output", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Description: ""})

		output := out.String()
		assert.NotContains(t, output, "Description")
	})

	t.Run("shows collapsed editor field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Editor: "vim"})

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Editor")
		assert.Contains(t, output, "vim")
	})

	t.Run("omits empty editor from output", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, createResult{Name: "my-project", Location: "/tmp/dev", Editor: ""})

		output := out.String()
		assert.NotContains(t, output, "Editor")
	})
}

func TestValidateCreateName(t *testing.T) {
	t.Run("empty string returns Name cannot be empty", func(t *testing.T) {
		err := validateCreateName("")
		assert.EqualError(t, err, "Name cannot be empty")
	})

	t.Run("whitespace-only returns Name cannot be empty", func(t *testing.T) {
		err := validateCreateName("   ")
		assert.EqualError(t, err, "Name cannot be empty")
	})

	t.Run("valid name returns nil", func(t *testing.T) {
		err := validateCreateName("my-project")
		assert.NoError(t, err)
	})
}

func TestHandleCreateFormError(t *testing.T) {
	t.Run("ErrUserAborted returns nil", func(t *testing.T) {
		err := handleCreateFormError(huh.ErrUserAborted)
		assert.NoError(t, err)
	})

	t.Run("other errors propagate", func(t *testing.T) {
		err := handleCreateFormError(errors.New("unexpected"))
		assert.Error(t, err)
	})
}
