package ui

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestWizardTheme(t *testing.T) {
	t.Run("error message renders with cross prefix", func(t *testing.T) {
		theme := WizardTheme()
		rendered := theme.Focused.ErrorMessage.Render("test error")
		stripped := stripANSI(rendered)
		assert.Contains(t, stripped, "✗ test error")
	})

	t.Run("blurred error message renders with cross prefix", func(t *testing.T) {
		theme := WizardTheme()
		rendered := theme.Blurred.ErrorMessage.Render("test error")
		stripped := stripANSI(rendered)
		assert.Contains(t, stripped, "✗ test error")
	})
}

func TestRenderSuccess(t *testing.T) {
	t.Run("shows created header with name", func(t *testing.T) {
		output := stripANSI(RenderSuccess("my-project", "/home/user/my-project", []string{"Directory created"}))

		assert.Contains(t, output, "◆ Created my-project")
	})

	t.Run("shows path on its own line", func(t *testing.T) {
		output := stripANSI(RenderSuccess("my-project", "/home/user/my-project", []string{"Directory created"}))

		assert.Contains(t, output, "/home/user/my-project")
	})

	t.Run("shows checkmarks for each check", func(t *testing.T) {
		checks := []string{"Directory created", "Git initialized", "Added to catalog"}
		output := stripANSI(RenderSuccess("my-project", "/tmp/my-project", checks))

		assert.Contains(t, output, "✓ Directory created")
		assert.Contains(t, output, "✓ Git initialized")
		assert.Contains(t, output, "✓ Added to catalog")
	})

	t.Run("uses box-drawing borders", func(t *testing.T) {
		output := stripANSI(RenderSuccess("proj", "/tmp/proj", []string{"Done"}))

		assert.Contains(t, output, "┌")
		assert.Contains(t, output, "│")
		assert.Contains(t, output, "└")
	})
}
