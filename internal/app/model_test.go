package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func sized(t *testing.T, w, h int) Model {
	t.Helper()
	next, cmd := New().Update(tea.WindowSizeMsg{Width: w, Height: h})
	if cmd != nil {
		t.Fatalf("WindowSizeMsg returned a non-nil cmd")
	}
	return next.(Model)
}

func TestQuitKeys(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"q", tea.KeyPressMsg{Code: 'q', Text: "q"}},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := New().Update(tt.key)
			if cmd == nil {
				t.Fatal("expected a quit cmd, got nil")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("cmd() = %T, want tea.QuitMsg", cmd())
			}
		})
	}
}

func TestOtherKeysDoNotQuit(t *testing.T) {
	tests := []struct {
		name string
		key  tea.KeyPressMsg
	}{
		{"Q", tea.KeyPressMsg{Code: 'Q', Text: "Q"}},
		{"x", tea.KeyPressMsg{Code: 'x', Text: "x"}},
		{"space", tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEscape}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := sized(t, 80, 24)
			before := m.View().Content
			next, cmd := m.Update(tt.key)
			if cmd != nil {
				t.Fatalf("expected nil cmd, got %T", cmd())
			}
			if after := next.View().Content; after != before {
				t.Fatalf("view changed:\nbefore: %q\nafter:  %q", before, after)
			}
		})
	}
}

func TestWindowSizeTracking(t *testing.T) {
	m := sized(t, 80, 24)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(Model)
	if got.width != 120 || got.height != 40 {
		t.Fatalf("size = %dx%d, want 120x40", got.width, got.height)
	}
	if !strings.Contains(got.View().Content, "120x40") {
		t.Fatalf("render does not use latest size: %q", got.View().Content)
	}
}

func TestViewUsesAltScreen(t *testing.T) {
	if !New().View().AltScreen {
		t.Fatal("AltScreen = false, want true")
	}
	if !sized(t, 80, 24).View().AltScreen {
		t.Fatal("AltScreen = false after resize, want true")
	}
}

func TestRenderFitsWindow(t *testing.T) {
	for w := 0; w <= 130; w++ {
		for h := 0; h <= 45; h++ {
			m := Model{width: w, height: h}
			content := m.View().Content
			lines := strings.Split(content, "\n")
			for _, l := range lines {
				if strings.HasSuffix(l, " ") {
					t.Fatalf("%dx%d: line %q ends in a space", w, h, l)
				}
			}
			if w <= 0 || h <= 0 {
				continue
			}
			if len(lines) > h {
				t.Fatalf("%dx%d: %d lines, want at most %d", w, h, len(lines), h)
			}
			for _, l := range lines {
				if len(l) > w {
					t.Fatalf("%dx%d: line %q is %d bytes, want at most %d", w, h, l, len(l), w)
				}
			}
		}
	}
}

func TestRenderCentered80x24(t *testing.T) {
	lines := strings.Split(sized(t, 80, 24).View().Content, "\n")
	// (24-3)/2 = 10 leading blank lines, then three text lines.
	if len(lines) != 13 {
		t.Fatalf("got %d lines, want 13", len(lines))
	}
	for i := 0; i < 10; i++ {
		if lines[i] != "" {
			t.Fatalf("line %d = %q, want blank", i, lines[i])
		}
	}
	want := []string{
		strings.Repeat(" ", 38) + "mhz",             // (80-3)/2
		strings.Repeat(" ", 32) + "press q to quit", // (80-15)/2
		strings.Repeat(" ", 37) + "80x24",           // (80-5)/2
	}
	for i, w := range want {
		if got := lines[10+i]; got != w {
			t.Errorf("line %d = %q, want %q", 10+i, got, w)
		}
	}
}

func TestRenderExactFit15x3(t *testing.T) {
	got := sized(t, 15, 3).View().Content
	// "mhz" is (15-3)/2 = 6 spaces in, "15x3" is (15-4)/2 = 5 spaces in.
	want := strings.Repeat(" ", 6) + "mhz\npress q to quit\n" + strings.Repeat(" ", 5) + "15x3"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderUnsized(t *testing.T) {
	got := New().View().Content
	want := "mhz\npress q to quit\n0x0"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
