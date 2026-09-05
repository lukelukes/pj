package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderReceipt(t *testing.T) {
	pal := plainPalette()

	t.Run("names the project and its path", func(t *testing.T) {
		out := stripANSI(RenderReceipt(Receipt{
			Name:  "api-gateway",
			Path:  "~/dev/api-gateway",
			Steps: []string{"dir", "git", "catalog"},
		}, pal))

		assert.Contains(t, out, "✓ api-gateway  ~/dev/api-gateway")
		assert.Contains(t, out, "dir · git · catalog")
	})

	t.Run("adopted projects say so", func(t *testing.T) {
		out := stripANSI(RenderReceipt(Receipt{
			Name:    "legacy",
			Path:    "~/dev/legacy",
			Steps:   []string{"catalog"},
			Adopted: true,
		}, pal))

		assert.Contains(t, out, "✓ adopted legacy")
	})

	t.Run("jumped adds the cd marker", func(t *testing.T) {
		out := stripANSI(RenderReceipt(Receipt{
			Name:   "x",
			Path:   "~/x",
			Steps:  []string{"dir"},
			Jumped: true,
		}, pal))

		assert.Contains(t, out, "dir · cd'd")
	})

	t.Run("no steps and no jump renders a single line", func(t *testing.T) {
		out := stripANSI(RenderReceipt(Receipt{Name: "x", Path: "~/x"}, pal))

		assert.Equal(t, 1, strings.Count(out, "\n"))
	})

	t.Run("never exceeds two lines", func(t *testing.T) {
		out := stripANSI(RenderReceipt(Receipt{
			Name:    "x",
			Path:    "~/x",
			Steps:   []string{"dir", "git", ".gitignore", "catalog"},
			Adopted: true,
			Jumped:  true,
		}, pal))

		require.LessOrEqual(t, strings.Count(strings.TrimRight(out, "\n"), "\n")+1, 2)
	})
}
