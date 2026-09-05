package wizardtea

import (
	"errors"
	"io"
	"pj/internal/ui"

	tea "charm.land/bubbletea/v2"
)

type Runner struct {
	Palette ui.Palette
	Input   io.Reader
	Output  io.Writer
}

func (r Runner) Run(s ui.Session) (ui.Outcome, error) {
	opts := make([]tea.ProgramOption, 0, 2)
	if r.Input != nil {
		opts = append(opts, tea.WithInput(r.Input))
	}
	if r.Output != nil {
		opts = append(opts, tea.WithOutput(r.Output))
	}

	final, err := tea.NewProgram(newModel(s, r.Palette), opts...).Run()
	if err != nil {
		if errors.Is(err, tea.ErrInterrupted) || errors.Is(err, tea.ErrProgramKilled) {
			return ui.Outcome{Cancelled: true}, nil
		}
		return ui.Outcome{}, err
	}

	m, ok := final.(*model)
	if !ok {
		return ui.Outcome{Cancelled: true}, nil
	}
	return m.outcome, nil
}
