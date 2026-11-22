package wizardtea

import (
	"pj/internal/ui"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type field int

const (
	fieldName field = iota
	fieldLocation
	fieldDescription
	fieldTags
	fieldEditor
	fieldGit
	fieldCount
)

type mode int

const (
	modeForm mode = iota
	modeConfirm
)

const textFields = 5

type model struct {
	session ui.Session
	pal     ui.Palette

	inputs [textFields]textinput.Model
	git    bool
	focus  field
	passed [fieldCount]bool
	mode   mode

	preview  ui.Preview
	outcome  ui.Outcome
	touched  bool
	finished bool
}

func newModel(s ui.Session, pal ui.Palette) *model {
	m := &model{session: s, pal: pal, git: s.Draft.Git}

	values := [textFields]string{
		s.Draft.Name,
		s.Draft.Location,
		s.Draft.Description,
		s.Draft.Tags,
		s.Draft.Editor,
	}
	placeholders := [textFields]string{"", "", "what is it for", "lang:go, cli", s.EditorHint}

	for i := range m.inputs {
		in := textinput.New()
		in.Prompt = ""
		in.SetValue(values[i])
		in.Placeholder = placeholders[i]
		in.SetWidth(inputWidth)
		m.inputs[i] = in
	}

	m.refresh()
	return m
}

func (m *model) draft() ui.Draft {
	return ui.Draft{
		Name:        m.inputs[fieldName].Value(),
		Location:    m.inputs[fieldLocation].Value(),
		Description: m.inputs[fieldDescription].Value(),
		Tags:        m.inputs[fieldTags].Value(),
		Editor:      m.inputs[fieldEditor].Value(),
		Git:         m.git,
	}
}

func (m *model) refresh() {
	m.preview = m.session.PreviewOf(m.draft())
}

func (m *model) Init() tea.Cmd {
	return m.focusOn(fieldName)
}

func (m *model) focusOn(f field) tea.Cmd {
	m.focus = f
	var cmd tea.Cmd
	for i := range m.inputs {
		if field(i) == f {
			cmd = m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()
	}
	return cmd
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	if m.mode == modeConfirm {
		return m, m.updateConfirm(key.String())
	}
	if cmd, handled := m.updateGlobal(key.String()); handled {
		return m, cmd
	}
	if m.focus == fieldGit {
		m.updateGit(key.String())
		return m, nil
	}

	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	m.touched = true
	m.refresh()
	return m, cmd
}

func (m *model) updateGlobal(k string) (tea.Cmd, bool) {
	switch k {
	case "ctrl+c":
		return m.cancel(), true
	case "esc":
		return m.cancel(), true
	case "enter":
		return m.advance(), true
	case "tab", "down":
		return m.move(1), true
	case "shift+tab", "up":
		return m.move(-1), true
	}
	return nil, false
}

func (m *model) cancel() tea.Cmd {
	m.outcome = ui.Outcome{Cancelled: true}
	m.finished = true
	return tea.Quit
}

func (m *model) updateConfirm(k string) tea.Cmd {
	switch k {
	case "y", "Y":
		m.outcome = ui.Outcome{Draft: m.draft(), Adopt: true}
		m.finished = true
		return tea.Quit
	case "ctrl+c":
		return m.cancel()
	default:
		m.mode = modeForm
		return nil
	}
}

func (m *model) updateGit(k string) {
	switch k {
	case "space", "left", "right", "h", "l", "y", "n":
		m.git = !m.git
		m.refresh()
	}
}

func (m *model) advance() tea.Cmd {
	if m.onLastField() {
		return m.commit()
	}
	return m.move(1)
}

func (m *model) onLastField() bool {
	return m.focus == fieldCount-1
}

func (m *model) commit() tea.Cmd {
	if m.preview.Blocked() {
		m.touched = true
		return nil
	}
	if m.preview.NeedsAdopt {
		m.mode = modeConfirm
		return nil
	}
	m.outcome = ui.Outcome{Draft: m.draft()}
	m.finished = true
	return tea.Quit
}

func (m *model) move(delta int) tea.Cmd {
	m.passed[m.focus] = true

	next := (int(m.focus) + delta + int(fieldCount)) % int(fieldCount)
	if field(next) == fieldGit {
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		m.focus = fieldGit
		return nil
	}
	return m.focusOn(field(next))
}
