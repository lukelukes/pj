package ui

import "strings"

const (
	factSeparator = " · "
	blockSymbol   = "✗"
	noticeSymbol  = "!"
)

func RenderPreview(p Preview, pal Palette) string {
	switch p.Severity {
	case SeverityBlock:
		return pal.Bad.Render(blockSymbol + " " + p.Message)
	case SeverityNotice:
		line := pal.Warn.Render(noticeSymbol + " " + p.Message)
		if p.NeedsAdopt && p.AdoptHint != "" {
			line += pal.Muted.Render("   " + p.AdoptHint)
		}
		return line
	default:
		return pal.Muted.Render(renderFacts(p))
	}
}

func renderFacts(p Preview) string {
	parts := make([]string, 0, len(p.Facts)+1)
	if p.Path != "" {
		parts = append(parts, p.Path)
	}
	parts = append(parts, p.Facts...)
	return strings.Join(parts, factSeparator)
}
