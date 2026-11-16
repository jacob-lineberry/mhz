// Package golden compares rendered output against checked-in golden files.
//
// Running tests with UPDATE_GOLDEN=1 in the environment rewrites the golden
// files instead of comparing against them. The variable is read on every
// assertion, never at package init: go test only tracks environment reads made
// after the test binary starts, so an init-time read could let
// "UPDATE_GOLDEN=1 go test ./..." return a cached pass without rewriting
// anything. No flag is registered, so the same command works for every
// package, whether or not it uses this helper.
package golden

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

const hint = "UPDATE_GOLDEN=1 go test ./..."

func updating() bool { return os.Getenv("UPDATE_GOLDEN") == "1" }

// RequireView sizes m with a tea.WindowSizeMsg, renders its View content and
// compares it with testdata/<test name>/<width>x<height>.golden. Commands
// returned by Update are not executed.
func RequireView(tb testing.TB, m tea.Model, width, height int) {
	tb.Helper()
	next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	path := filepath.Join("testdata", sanitize(tb.Name()), fmt.Sprintf("%dx%d.golden", width, height))
	RequireEqual(tb, path, []byte(next.View().Content))
}

// RequireEqual compares got with the file at path byte for byte. In update
// mode it writes the file (creating directories) instead.
func RequireEqual(tb testing.TB, path string, got []byte) {
	tb.Helper()

	if updating() {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			tb.Fatalf("golden: creating directory for %s: %v", path, err)
			return
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			tb.Fatalf("golden: writing %s: %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("golden: file %s does not exist; create it with: %s", path, hint)
		return
	}
	if err != nil {
		tb.Fatalf("golden: reading %s: %v", path, err)
		return
	}
	if bytes.Equal(want, got) {
		return
	}

	line, wantLine, gotLine := firstDiff(string(want), string(got))
	tb.Fatalf("golden: %s differs at line %d\n  want: %s\n  got:  %s\nupdate with: %s",
		path, line, wantLine, gotLine, hint)
}

// firstDiff returns the 1-based number of the first line that differs, and
// that line from each side already passed through strconv.Quote, so ANSI
// escapes are visible. A side that has run out of lines is shown as <none>,
// which keeps a missing line (for example a missing trailing newline)
// distinct from an empty one.
func firstDiff(want, got string) (line int, wantLine, gotLine string) {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	show := func(lines []string, i int) string {
		if i >= len(lines) {
			return "<none>"
		}
		return strconv.Quote(lines[i])
	}
	for i := range max(len(w), len(g)) {
		if (i < len(w)) != (i < len(g)) || w[i] != g[i] {
			return i + 1, show(w, i), show(g, i)
		}
	}
	return 0, "", "" // unreachable: split is lossless, so equal lines mean equal bytes
}

// sanitize maps every rune outside [A-Za-z0-9._-] to '_', including the '/'
// between subtest names, so the result is a valid single path element on
// every OS.
func sanitize(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			return r
		}
		return '_'
	}, name)
}
