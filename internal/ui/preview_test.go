package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderPreview(t *testing.T) {
	pal := plainPalette()

	t.Run("ok state joins path and facts", func(t *testing.T) {
		out := stripANSI(RenderPreview(Preview{
			Path:  "~/dev/api-gateway",
			Facts: []string{"git init", "nvim"},
		}, pal))

		assert.Equal(t, "~/dev/api-gateway · git init · nvim", out)
	})

	t.Run("ok state with no facts shows only the path", func(t *testing.T) {
		out := stripANSI(RenderPreview(Preview{Path: "~/dev/x"}, pal))

		assert.Equal(t, "~/dev/x", out)
	})

	t.Run("block state replaces the line with the message", func(t *testing.T) {
		out := stripANSI(RenderPreview(Preview{
			Path:     "~/dev/booster",
			Facts:    []string{"git init"},
			Severity: SeverityBlock,
			Message:  `~/dev/booster is already tracked as "booster"`,
		}, pal))

		assert.Equal(t, `✗ ~/dev/booster is already tracked as "booster"`, out)
		assert.NotContains(t, out, "git init", "a blocked preview must not advertise work it will not do")
	})

	t.Run("notice state shows the adopt hint", func(t *testing.T) {
		out := stripANSI(RenderPreview(Preview{
			Severity:   SeverityNotice,
			Message:    "~/dev/x exists · 12 files",
			NeedsAdopt: true,
			AdoptHint:  "↵ adopt into catalog",
		}, pal))

		assert.Contains(t, out, "! ~/dev/x exists · 12 files")
		assert.Contains(t, out, "↵ adopt into catalog")
	})

	t.Run("notice without adopt omits the hint", func(t *testing.T) {
		out := stripANSI(RenderPreview(Preview{
			Severity: SeverityNotice,
			Message:  "name already used by ~/dev/other",
		}, pal))

		assert.Equal(t, "! name already used by ~/dev/other", out)
	})
}

func TestPreviewBlocked(t *testing.T) {
	assert.False(t, Preview{Severity: SeverityOK}.Blocked())
	assert.False(t, Preview{Severity: SeverityNotice}.Blocked())
	assert.True(t, Preview{Severity: SeverityBlock}.Blocked())
}

func TestRenderPreviewIsSingleLine(t *testing.T) {
	pal := plainPalette()
	previews := []Preview{
		{Path: "~/a", Facts: []string{"git init", "nvim", "inside ~/mono"}},
		{Severity: SeverityNotice, Message: "exists", NeedsAdopt: true, AdoptHint: "↵ adopt"},
		{Severity: SeverityBlock, Message: "boom"},
		{},
	}

	for _, p := range previews {
		assert.NotContains(t, RenderPreview(p, pal), "\n",
			"the preview must stay on one line so the wizard height never moves")
	}
}

func TestSessionPreviewOfToleratesNilFunc(t *testing.T) {
	assert.Equal(t, Preview{}, Session{}.PreviewOf(Draft{Name: "x"}))
}
