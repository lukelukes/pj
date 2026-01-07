package render

import (
	"io"
	"os"
	"pj/internal/config"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
)

const staleThreshold = 30 * 24 * time.Hour

type LipglossRenderer struct {
	width int
	now   func() time.Time
	r     *lipgloss.Renderer

	nameStyle       lipgloss.Style
	pathStyle       lipgloss.Style
	descStyle       lipgloss.Style
	timeStyle       lipgloss.Style
	staleStyle      lipgloss.Style
	recentTimeStyle lipgloss.Style
}

func NewLipglossRenderer(w io.Writer, width int) *LipglossRenderer {
	r := lipgloss.NewRenderer(w)
	return &LipglossRenderer{
		width:           width,
		now:             time.Now,
		r:               r,
		nameStyle:       r.NewStyle().Bold(true),
		pathStyle:       r.NewStyle().Faint(true),
		descStyle:       r.NewStyle(),
		timeStyle:       r.NewStyle().Faint(true),
		recentTimeStyle: r.NewStyle().Foreground(lipgloss.Color("10")),
		staleStyle:      r.NewStyle().Faint(true),
	}
}

func NewLipglossRendererAuto(w io.Writer) *LipglossRenderer {
	width := 80
	if f, ok := w.(*os.File); ok {
		if tw, _, err := term.GetSize(f.Fd()); err == nil && tw > 0 {
			width = tw
		}
	}
	return NewLipglossRenderer(w, width)
}

func (r *LipglossRenderer) WithClock(now func() time.Time) *LipglossRenderer {
	r.now = now
	return r
}

func (r *LipglossRenderer) RenderProjectList(view ProjectListView) string {
	if view.IsEmpty() {
		return "No projects found.\n"
	}

	now := r.now()
	var sb strings.Builder
	for i, item := range view.Items {
		last := i == len(view.Items)-1
		sb.WriteString(r.renderItem(item, now, last))
	}
	sb.WriteString("\n")
	return sb.String()
}

func (r *LipglossRenderer) renderItem(item ProjectListItem, now time.Time, last bool) string {
	age := now.Sub(item.Timestamp)
	isStale := age > staleThreshold
	timeStr := r.formatTime(item.Timestamp, now)

	nameStyle := r.nameStyle
	pathStyle := r.pathStyle
	descStyle := r.descStyle
	timeStyle := r.timeStyle
	if isStale {
		nameStyle = r.staleStyle.Bold(true)
		pathStyle = r.staleStyle
		descStyle = r.staleStyle
		timeStyle = r.staleStyle
	} else if age < 1*time.Hour {
		timeStyle = r.recentTimeStyle
	}

	name := nameStyle.Render(item.Name)
	path := pathStyle.Render("  " + config.ShortenPath(item.Path))
	timeEl := timeStyle.Render(timeStr)

	padding := max(1, r.width-lipgloss.Width(name)-lipgloss.Width(timeEl))
	headerLine := name + strings.Repeat(" ", padding) + timeEl

	var lines []string
	lines = append(lines, headerLine)
	lines = append(lines, path)
	if item.Description != "" {
		desc := descStyle.Render("  " + item.Description)
		lines = append(lines, desc)
	}
	if !last {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (r *LipglossRenderer) formatTime(t, now time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}

	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	target := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	days := int(today.Sub(target).Hours() / 24)

	timeStr := t.Format("15:04")

	switch {
	case days == 0:
		return timeStr
	case days == 1:
		return "Yesterday " + timeStr
	case days < 7:
		return t.Format("Mon") + " " + timeStr
	case t.Year() == now.Year():
		return t.Format("Jan 2") + " " + timeStr
	default:
		return t.Format("Jan 2 '06") + " " + timeStr
	}
}
