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

func TestRenderWizard(t *testing.T) {
	t.Run("completed field renders collapsed with value", func(t *testing.T) {
		fields := []Field{{Label: "Name", Value: "my-project"}}
		output := stripANSI(RenderWizard("Title", fields, -1))

		assert.Contains(t, output, "◇ Name · my-project")
	})

	t.Run("active field renders with diamond and no value", func(t *testing.T) {
		fields := []Field{{Label: "Name"}}
		output := stripANSI(RenderWizard("Title", fields, 0))

		assert.Contains(t, output, "◆ Name")
		assert.NotContains(t, output, separator)
	})

	t.Run("optional active field renders with optional suffix", func(t *testing.T) {
		fields := []Field{{Label: "Desc", Optional: true}}
		output := stripANSI(RenderWizard("Title", fields, 0))

		assert.Contains(t, output, "◆ Desc (optional)")
	})

	t.Run("all fields completed renders without side border spacer", func(t *testing.T) {
		fields := []Field{
			{Label: "Name", Value: "my-project"},
			{Label: "Path", Value: "/home/user/projects/my-project"},
		}
		output := stripANSI(RenderWizard("Create", fields, -1))

		assert.Contains(t, output, "◇ Name · my-project")
		assert.Contains(t, output, "◇ Path · /home/user/projects/my-project")
	})

	t.Run("title renders after top border", func(t *testing.T) {
		fields := []Field{{Label: "Name", Value: "x"}}
		output := stripANSI(RenderWizard("Create new project", fields, -1))

		assert.Contains(t, output, "┌ Create new project")
	})

	t.Run("bottom border present", func(t *testing.T) {
		fields := []Field{{Label: "Name", Value: "x"}}
		output := stripANSI(RenderWizard("Title", fields, -1))

		assert.Contains(t, output, "└")
	})

	t.Run("empty-value non-active field produces no output line", func(t *testing.T) {
		fields := []Field{
			{Label: "Name", Value: "x"},
			{Label: "Empty"},
		}
		output := stripANSI(RenderWizard("Title", fields, -1))

		assert.NotContains(t, output, "Empty")
	})
}
