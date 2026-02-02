package ui

import (
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	activeSymbol   = "◆"
	completeSymbol = "◇"
	separator      = " · "
	borderTop      = "┌"
	borderSide     = "│"
	borderBottom   = "└"
	checkSymbol    = "✓"
)

func WizardTheme() *huh.Theme {
	t := huh.ThemeBase()
	red := lipgloss.Color("1")
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.SetString("✗").Foreground(red)
	t.Blurred.ErrorMessage = t.Blurred.ErrorMessage.SetString("✗").Foreground(red)
	return t
}

type Field struct {
	Label    string
	Value    string
	Optional bool
}

func borderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
}

func RenderWizard(title string, fields []Field, activeIdx int) string {
	var b strings.Builder

	border := borderStyle()

	b.WriteString(border.Render(borderTop))
	b.WriteString(" ")
	b.WriteString(title)
	b.WriteString("\n")

	b.WriteString(border.Render(borderSide))
	b.WriteString("\n")

	for i, f := range fields {
		active := i == activeIdx
		if f.Value != "" || active {
			b.WriteString(renderField(f, active))
			b.WriteString("\n")
		}
	}

	if activeIdx >= 0 && activeIdx < len(fields) {
		b.WriteString(border.Render(borderSide))
		b.WriteString("\n")
	}

	b.WriteString(border.Render(borderBottom))
	b.WriteString("\n")

	return b.String()
}

func RenderSuccess(name, path string, checks []string) string {
	var b strings.Builder

	border := borderStyle()

	b.WriteString(border.Render(borderTop))
	b.WriteString(" ")
	b.WriteString(activeSymbol)
	b.WriteString(" Created ")
	b.WriteString(name)
	b.WriteString("\n")

	b.WriteString(border.Render(borderSide))
	b.WriteString(" ")
	b.WriteString(path)
	b.WriteString("\n")

	b.WriteString(border.Render(borderSide))
	b.WriteString("\n")

	for _, check := range checks {
		b.WriteString(border.Render(borderSide))
		b.WriteString(" ")
		b.WriteString(checkSymbol)
		b.WriteString(" ")
		b.WriteString(check)
		b.WriteString("\n")
	}

	b.WriteString(border.Render(borderBottom))
	b.WriteString("\n")

	return b.String()
}

func renderField(f Field, active bool) string {
	var b strings.Builder

	if active {
		b.WriteString(activeSymbol)
		b.WriteString(" ")
		b.WriteString(f.Label)
		if f.Optional {
			b.WriteString(" (optional)")
		}
	} else {
		b.WriteString(completeSymbol)
		b.WriteString(" ")
		b.WriteString(f.Label)
		b.WriteString(separator)
		b.WriteString(f.Value)
	}

	return b.String()
}
