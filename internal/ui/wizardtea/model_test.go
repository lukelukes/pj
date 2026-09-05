package wizardtea

import (
	"pj/internal/ui"
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tea "charm.land/bubbletea/v2"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func plainLines(s string) string {
	lines := strings.Split(stripANSI(s), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func plainPalette() ui.Palette {
	s := lipgloss.NewStyle()
	return ui.Palette{Normal: s, Muted: s, Strong: s, Accent: s, Good: s, Warn: s, Bad: s}
}

func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	default:
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
}

func send(t *testing.T, m *model, keys ...string) *model {
	t.Helper()
	for _, k := range keys {
		next, _ := m.Update(keyPress(k))
		updated, ok := next.(*model)
		require.True(t, ok)
		m = updated
	}
	return m
}

func commit(t *testing.T, m *model) *model {
	t.Helper()
	return send(t, m, "shift+tab", "enter")
}

func typeText(t *testing.T, m *model, text string) *model {
	t.Helper()
	for _, r := range text {
		m = send(t, m, string(r))
	}
	return m
}

type stubPreview struct {
	blockNames map[string]string
	adoptNames map[string]string
}

func (s stubPreview) fn(d ui.Draft) ui.Preview {
	if msg, ok := s.blockNames[d.Name]; ok {
		return ui.Preview{Severity: ui.SeverityBlock, Message: msg}
	}
	if msg, ok := s.adoptNames[d.Name]; ok {
		return ui.Preview{
			Severity:   ui.SeverityNotice,
			Message:    msg,
			NeedsAdopt: true,
			AdoptHint:  "↵ adopt into catalog",
		}
	}
	if d.Name == "" {
		return ui.Preview{Severity: ui.SeverityBlock, Message: "name is required"}
	}
	p := ui.Preview{Path: d.Location + "/" + d.Name}
	if d.Git {
		p.Facts = append(p.Facts, "git init")
	}
	return p
}

func newTestModel(t *testing.T, stub stubPreview) *model {
	t.Helper()
	m := newModel(ui.Session{
		Draft:      ui.Draft{Location: "~/dev", Git: true},
		Preview:    stub.fn,
		EditorHint: "nvim",
	}, plainPalette())
	m.Init()
	return m
}

func TestModelStartsOnTheNameFieldWithEveryStepVisible(t *testing.T) {
	m := newTestModel(t, stubPreview{})

	assert.Equal(t, modeForm, m.mode)
	assert.Equal(t, fieldName, m.focus)
	for f := range fieldCount {
		assert.Contains(t, plainLines(m.render()), labels[f]+":", "every step is on screen from the start")
	}
}

func TestStepsTickOnlyOnceYouLeaveThem(t *testing.T) {
	m := newTestModel(t, stubPreview{})

	assert.NotContains(t, plainLines(m.render()), markerDone, "an unconfirmed default is not a done step")

	m = send(t, typeText(t, m, "api"), "enter")

	assert.Contains(t, plainLines(m.render()), markerDone+"  name:")
	assert.NotContains(t, plainLines(m.render()), markerDone+"  where:", "the step you are on is never ticked")
}

func TestSkippedEmptyStepsStayUnticked(t *testing.T) {
	m := send(t, typeText(t, newTestModel(t, stubPreview{}), "api"), "enter", "enter", "enter")

	assert.Equal(t, fieldEditor, m.focus)
	assert.NotContains(t, plainLines(m.render()), markerDone+"  about:", "a step left blank is not done")
}

func TestEnterAdvancesUntilTheLastField(t *testing.T) {
	t.Run("enter walks down the fields", func(t *testing.T) {
		m := typeText(t, newTestModel(t, stubPreview{}), "api")

		for _, want := range []field{fieldLocation, fieldDescription, fieldEditor, fieldGit} {
			m = send(t, m, "enter")
			require.Equal(t, want, m.focus)
			require.False(t, m.finished, "enter must not create from the middle of the form")
		}
	})

	t.Run("enter on the last field creates", func(t *testing.T) {
		m := typeText(t, newTestModel(t, stubPreview{}), "api")
		m = send(t, m, "shift+tab")
		require.Equal(t, fieldGit, m.focus)

		m = send(t, m, "enter")

		assert.True(t, m.finished)
		assert.Equal(t, "api", m.outcome.Draft.Name)
	})
}

func TestTypingUpdatesThePreviewLive(t *testing.T) {
	m := newTestModel(t, stubPreview{})

	m = typeText(t, m, "api")

	assert.Equal(t, "~/dev/api", m.preview.Path)
	assert.Equal(t, []string{"git init"}, m.preview.Facts)
}

func TestEnterCommitsAValidDraft(t *testing.T) {
	m := newTestModel(t, stubPreview{})
	m = typeText(t, m, "api")

	m = commit(t, m)

	assert.True(t, m.finished)
	assert.False(t, m.outcome.Cancelled)
	assert.False(t, m.outcome.Adopt)
	assert.Equal(t, "api", m.outcome.Draft.Name)
	assert.Equal(t, "~/dev", m.outcome.Draft.Location)
	assert.True(t, m.outcome.Draft.Git)
}

func TestEnterIsInertWhileBlocked(t *testing.T) {
	m := newTestModel(t, stubPreview{blockNames: map[string]string{"booster": "name taken"}})
	m = typeText(t, m, "booster")

	m = commit(t, m)

	assert.False(t, m.finished, "a blocked draft must never commit")
}

func TestEmptyNameCannotCommit(t *testing.T) {
	m := commit(t, newTestModel(t, stubPreview{}))

	assert.False(t, m.finished)
}

func TestAdoptionRequiresExplicitConfirmation(t *testing.T) {
	stub := stubPreview{adoptNames: map[string]string{"legacy": "~/dev/legacy exists · 12 files"}}

	t.Run("enter opens the confirmation instead of committing", func(t *testing.T) {
		m := commit(t, typeText(t, newTestModel(t, stub), "legacy"))

		assert.Equal(t, modeConfirm, m.mode)
		assert.False(t, m.finished, "enter alone must never adopt an existing directory")
	})

	t.Run("y confirms the adoption", func(t *testing.T) {
		m := commit(t, typeText(t, newTestModel(t, stub), "legacy"))

		m = send(t, m, "y")

		assert.True(t, m.finished)
		assert.True(t, m.outcome.Adopt)
		assert.Equal(t, "legacy", m.outcome.Draft.Name)
	})

	t.Run("a second enter goes back rather than confirming", func(t *testing.T) {
		m := commit(t, typeText(t, newTestModel(t, stub), "legacy"))

		m = send(t, m, "enter")

		assert.False(t, m.finished)
		assert.False(t, m.outcome.Adopt)
		assert.Equal(t, modeForm, m.mode)
	})

	t.Run("any other key backs out", func(t *testing.T) {
		for _, k := range []string{"n", "esc", "q", "tab"} {
			m := commit(t, typeText(t, newTestModel(t, stub), "legacy"))

			m = send(t, m, k)

			assert.False(t, m.outcome.Adopt, "%q must not confirm adoption", k)
			assert.Equal(t, modeForm, m.mode)
		}
	})

	t.Run("ctrl+c during confirmation cancels everything", func(t *testing.T) {
		m := commit(t, typeText(t, newTestModel(t, stub), "legacy"))

		m = send(t, m, "ctrl+c")

		assert.True(t, m.finished)
		assert.True(t, m.outcome.Cancelled)
		assert.False(t, m.outcome.Adopt)
	})
}

func TestCancellation(t *testing.T) {
	t.Run("esc cancels from any field", func(t *testing.T) {
		for _, prefix := range [][]string{{}, {"tab"}, {"tab", "tab"}, {"shift+tab"}} {
			m := send(t, newTestModel(t, stubPreview{}), append(prefix, "esc")...)

			assert.True(t, m.finished)
			assert.True(t, m.outcome.Cancelled)
		}
	})

	t.Run("ctrl+c always cancels", func(t *testing.T) {
		for _, prefix := range [][]string{{}, {"tab"}, {"tab", "tab"}} {
			m := send(t, newTestModel(t, stubPreview{}), append(prefix, "ctrl+c")...)

			assert.True(t, m.finished)
			assert.True(t, m.outcome.Cancelled)
		}
	})
}

func TestNavigation(t *testing.T) {
	t.Run("tab moves to the next field", func(t *testing.T) {
		m := send(t, newTestModel(t, stubPreview{}), "tab")

		assert.Equal(t, fieldLocation, m.focus)
	})

	t.Run("shift+tab wraps backwards onto git", func(t *testing.T) {
		m := send(t, newTestModel(t, stubPreview{}), "shift+tab")

		assert.Equal(t, fieldGit, m.focus)
	})

	t.Run("tab wraps around every field", func(t *testing.T) {
		m := newTestModel(t, stubPreview{})
		want := []field{fieldLocation, fieldDescription, fieldEditor, fieldGit, fieldName}

		for _, expected := range want {
			m = send(t, m, "tab")
			assert.Equal(t, expected, m.focus)
		}
	})

	t.Run("only the focused input is editable", func(t *testing.T) {
		m := send(t, newTestModel(t, stubPreview{}), "tab")
		m = typeText(t, m, "x")

		assert.Empty(t, m.inputs[fieldName].Value())
		assert.Equal(t, "~/devx", m.inputs[fieldLocation].Value())
	})
}

func TestGitToggle(t *testing.T) {
	m := send(t, newTestModel(t, stubPreview{}), "shift+tab")
	require.Equal(t, fieldGit, m.focus)
	require.True(t, m.git)

	m = send(t, m, "space")

	assert.False(t, m.git)
	assert.False(t, m.draft().Git)

	m = send(t, m, "space")
	assert.True(t, m.git)
}

func TestFinishedModelRendersNothing(t *testing.T) {
	m := commit(t, typeText(t, newTestModel(t, stubPreview{}), "api"))

	assert.Empty(t, m.render(), "the wizard must leave no frame behind once it commits")
}
