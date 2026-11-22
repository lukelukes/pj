package wizardtea

import (
	"pj/internal/ui"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	inputWidth  = 30
	valueWidth  = inputWidth + 1
	labelWidth  = 10
	markerDone  = "✓"
	markerFocus = "→"
	markerRow   = "│"
	markerFoot  = "└"
	markerGap   = "  "
)

var labels = [fieldCount]string{"name", "where", "about", "tags", "editor", "git"}

type rowState int

const (
	rowPending rowState = iota
	rowCurrent
	rowDone
)

func (m *model) View() tea.View {
	return tea.NewView(m.render())
}

func (m *model) render() string {
	if m.finished {
		return ""
	}
	if m.mode == modeConfirm {
		return trimLines(m.renderConfirm())
	}
	return trimLines(m.renderForm())
}

func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func (m *model) renderForm() string {
	var b strings.Builder

	for f := range fieldCount {
		state := m.state(f)
		b.WriteString(m.marker(state))
		b.WriteString(m.label(f, state))
		b.WriteString(m.value(f))
		if hint := m.hint(f); hint != "" {
			b.WriteString(m.pal.Muted.Render("  " + hint))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.pal.Muted.Render(markerRow) + "\n")
	b.WriteString(m.pal.Muted.Render(markerFoot) + markerGap + m.previewLine())
	b.WriteString("\n\n\n")

	b.WriteString(m.help("↵ next", "⇧⇥ prev", "esc cancel"))
	b.WriteString("\n")

	return b.String()
}

func (m *model) renderConfirm() string {
	var b strings.Builder

	b.WriteString(m.pal.Warn.Render("! " + m.preview.Message))
	b.WriteString("\n")

	b.WriteString("  ")
	b.WriteString(m.pal.Normal.Render("adopt it into the catalog?"))
	b.WriteString("  ")
	b.WriteString(m.help("y confirm", "any other key goes back"))
	b.WriteString("\n")

	return b.String()
}

func (m *model) previewLine() string {
	if !m.touched && m.preview.Blocked() {
		return m.pal.Muted.Render(m.pendingPath())
	}
	return ui.RenderPreview(m.preview, m.pal)
}

func (m *model) pendingPath() string {
	location := strings.TrimSuffix(m.inputs[fieldLocation].Value(), "/")
	if location == "" {
		return "…"
	}
	return location + "/…"
}

func (m *model) state(f field) rowState {
	switch {
	case m.focus == f:
		return rowCurrent
	case m.passed[f] && m.filled(f):
		return rowDone
	default:
		return rowPending
	}
}

func (m *model) filled(f field) bool {
	switch f {
	case fieldGit:
		return true
	case fieldEditor:
		return m.inputs[fieldEditor].Value() != "" || m.session.EditorHint != ""
	default:
		return m.inputs[f].Value() != ""
	}
}

func (m *model) style(state rowState) lipgloss.Style {
	switch state {
	case rowDone:
		return m.pal.Good
	case rowCurrent:
		return m.pal.Accent
	default:
		return m.pal.Muted
	}
}

func (m *model) marker(state rowState) string {
	symbol := markerRow
	switch state {
	case rowDone:
		symbol = markerDone
	case rowCurrent:
		symbol = markerFocus
	}
	return m.style(state).Render(symbol) + markerGap
}

func (m *model) label(f field, state rowState) string {
	name := labels[f] + ":"
	return m.style(state).Render(name + strings.Repeat(" ", max(1, labelWidth-len(name))))
}

func (m *model) value(f field) string {
	if f == fieldGit {
		return m.gitToggle()
	}
	return m.inputs[f].View()
}

func (m *model) gitToggle() string {
	init, skip := "○ init", "○ skip"
	if m.git {
		init = "● init"
	} else {
		skip = "● skip"
	}

	on, off := m.pal.Normal, m.pal.Muted
	if m.git {
		return padTo(on.Render(init)+"  "+off.Render(skip), valueWidth)
	}
	return padTo(off.Render(init)+"  "+on.Render(skip), valueWidth)
}

func padTo(s string, width int) string {
	if visible := lipgloss.Width(s); visible < width {
		return s + strings.Repeat(" ", width-visible)
	}
	return s
}

func (m *model) hint(f field) string {
	switch f {
	case fieldEditor:
		if m.inputs[fieldEditor].Value() == "" && m.session.EditorHint != "" {
			return "inherited"
		}
	case fieldGit:
		if m.focus == f && !m.preview.Blocked() {
			return "↵ creates"
		}
	}
	return ""
}

func (m *model) help(items ...string) string {
	return m.pal.Muted.Render(strings.Join(items, "   "))
}
