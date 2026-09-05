package ui

type Draft struct {
	Name        string
	Location    string
	Description string
	Editor      string
	Git         bool
}

type Severity int

const (
	SeverityOK Severity = iota
	SeverityNotice
	SeverityBlock
)

type Preview struct {
	Path       string
	Facts      []string
	Message    string
	Severity   Severity
	NeedsAdopt bool
	AdoptHint  string
}

func (p Preview) Blocked() bool { return p.Severity == SeverityBlock }

type PreviewFunc func(Draft) Preview

type Session struct {
	Draft      Draft
	Preview    PreviewFunc
	EditorHint string
	Home       string
}

func (s Session) PreviewOf(d Draft) Preview {
	if s.Preview == nil {
		return Preview{}
	}
	return s.Preview(d)
}

type Outcome struct {
	Draft     Draft
	Adopt     bool
	Cancelled bool
}

type Runner interface {
	Run(Session) (Outcome, error)
}
