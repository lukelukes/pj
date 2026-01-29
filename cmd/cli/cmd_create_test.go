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

		renderCreateSummary(g, "my-project")

		output := out.String()
		assert.Contains(t, output, "◇")
		assert.Contains(t, output, "Name")
		assert.Contains(t, output, "my-project")
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
