package ui

import "strings"

const (
	receiptSymbol = "✓"
	receiptIndent = "  "
)

type Receipt struct {
	Name    string
	Path    string
	Steps   []string
	Adopted bool
	Jumped  bool
}

func RenderReceipt(r Receipt, pal Palette) string {
	var b strings.Builder

	verb := receiptSymbol
	if r.Adopted {
		verb = receiptSymbol + " adopted"
	}

	b.WriteString(pal.Good.Render(verb))
	b.WriteString(" ")
	b.WriteString(pal.Strong.Render(r.Name))
	b.WriteString("  ")
	b.WriteString(pal.Muted.Render(r.Path))
	b.WriteString("\n")

	if len(r.Steps) > 0 || r.Jumped {
		b.WriteString(receiptIndent)
		b.WriteString(pal.Muted.Render(strings.Join(r.Steps, factSeparator)))
		if r.Jumped {
			b.WriteString(pal.Muted.Render(factSeparator + "cd'd"))
		}
		b.WriteString("\n")
	}

	return b.String()
}
