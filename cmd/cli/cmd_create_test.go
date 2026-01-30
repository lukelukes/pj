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

		renderCreateSummary(g, "my-project", "/home/user/projects")

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "my-project")
	})

	t.Run("shows collapsed location field", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, "my-project", "/home/user/projects")

		output := out.String()
		assert.Contains(t, output, "Location")
		assert.Contains(t, output, "/home/user/projects")
	})

	t.Run("shows both name and location fields", func(t *testing.T) {
		g, out := newTestGlobals(t)

		renderCreateSummary(g, "my-project", "/tmp/dev")

		output := out.String()
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "my-project")
		assert.Contains(t, output, "Location")
		assert.Contains(t, output, "/tmp/dev")
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
