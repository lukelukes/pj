package ui

import (
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const (
	activeSymbol = "◆"
	borderTop    = "┌"
	borderSide   = "│"
	borderBottom = "└"
	checkSymbol  = "✓"
)

func WizardTheme() *huh.Theme {
	t := huh.ThemeBase()
	red := lipgloss.Color("1")
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.SetString("✗").Foreground(red)
	t.Blurred.ErrorMessage = t.Blurred.ErrorMessage.SetString("✗").Foreground(red)
	return t
}

func RenderSuccess(name, path string, checks []string) string {
	var b strings.Builder

	border := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

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
