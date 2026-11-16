// Package app holds the top-level Bubble Tea model for mhz.
package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Model is the placeholder full-screen model. It only tracks the window size
// and quits on "q" or "ctrl+c".
type Model struct {
	width, height int
}

// New returns a Model that has not yet received a window size.
func New() Model { return Model{} }

// Init implements tea.Model.
func (Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model. The alt screen is requested on the View itself.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}

// render lays the placeholder out for the current window size.
//
// With no size report yet (width or height <= 0) the lines are joined without
// padding or clipping. Otherwise the block is centered with leading blank
// lines and leading spaces only, then cut to fit. Output never has trailing
// whitespace or trailing blank lines, so golden files read cleanly.
func (m Model) render() string {
	lines := []string{"mhz", "press q to quit", fmt.Sprintf("%dx%d", m.width, m.height)}
	if m.width <= 0 || m.height <= 0 {
		return strings.Join(lines, "\n")
	}

	top := max((m.height-len(lines))/2, 0)
	out := make([]string, 0, m.height)
	for range top {
		out = append(out, "")
	}
	for _, l := range lines {
		left := max((m.width-len(l))/2, 0)
		l = strings.Repeat(" ", left) + l
		// len and slicing count bytes, which equals terminal cells only
		// because the placeholder text is ASCII. The first change that renders
		// styled or non-ASCII text must switch to cell-width functions.
		if len(l) > m.width {
			l = l[:m.width]
		}
		out = append(out, strings.TrimRight(l, " "))
	}
	if len(out) > m.height {
		out = out[:m.height]
	}
	return strings.Join(out, "\n")
}
