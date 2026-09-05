package ui

import (
	"regexp"

	"charm.land/lipgloss/v2"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func plainPalette() Palette {
	s := lipgloss.NewStyle()
	return Palette{Normal: s, Muted: s, Strong: s, Accent: s, Good: s, Warn: s, Bad: s}
}
