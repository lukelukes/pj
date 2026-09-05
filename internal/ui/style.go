package ui

import "charm.land/lipgloss/v2"

type Palette struct {
	Normal lipgloss.Style
	Muted  lipgloss.Style
	Strong lipgloss.Style
	Accent lipgloss.Style
	Good   lipgloss.Style
	Warn   lipgloss.Style
	Bad    lipgloss.Style
}

func DefaultPalette() Palette {
	return Palette{
		Normal: lipgloss.NewStyle(),
		Muted:  lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		Strong: lipgloss.NewStyle().Bold(true),
		Accent: lipgloss.NewStyle().Foreground(lipgloss.Color("12")),
		Good:   lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		Warn:   lipgloss.NewStyle().Foreground(lipgloss.Color("11")),
		Bad:    lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
	}
}
