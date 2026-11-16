package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Tests here change the working directory and environment, so none of them
// call t.Parallel.

// stubTB records failures instead of failing the real test. It embeds the
// interface only to satisfy testing.TB's private method; the helper must not
// call anything beyond what is overridden here.
type stubTB struct {
	testing.TB
	name string
	msgs []string
}

func (s *stubTB) Helper()      {}
func (s *stubTB) Name() string { return s.name }
func (s *stubTB) Fatalf(format string, args ...any) {
	s.msgs = append(s.msgs, fmt.Sprintf(format, args...))
}
func (s *stubTB) Errorf(format string, args ...any) {
	s.msgs = append(s.msgs, fmt.Sprintf(format, args...))
}
func (s *stubTB) failed() bool   { return len(s.msgs) > 0 }
func (s *stubTB) output() string { return strings.Join(s.msgs, "\n") }

// sizeModel renders the size it was last given, so tests can tell that
// RequireView uses the model returned by Update.
type sizeModel struct{ w, h int }

func (sizeModel) Init() tea.Cmd { return nil }
func (m sizeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if s, ok := msg.(tea.WindowSizeMsg); ok {
		m.w, m.h = s.Width, s.Height
	}
	return m, nil
}
func (m sizeModel) View() tea.View { return tea.NewView(fmt.Sprintf("size %dx%d\nsecond", m.w, m.h)) }

// inTempDir switches to a fresh temp directory and sets UPDATE_GOLDEN.
func inTempDir(t *testing.T, update string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("UPDATE_GOLDEN", update)
	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// makeReadOnly makes any write to path fail, so "compare mode never touches
// the file" is enforced by the OS rather than inferred from timestamps. (The
// content checks that follow still guard the case where the tests run as
// root, which ignores file permissions.)
func makeReadOnly(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMatchPassesAndLeavesFileAlone(t *testing.T) {
	inTempDir(t, "")
	path := filepath.Join("testdata", "x.golden")
	writeFile(t, path, "hello\nworld")
	makeReadOnly(t, path)

	stub := &stubTB{name: "TestX"}
	RequireEqual(stub, path, []byte("hello\nworld"))

	if stub.failed() {
		t.Fatalf("unexpected failure (a write attempt fails on a read-only file): %s", stub.output())
	}
	if got := readFile(t, path); got != "hello\nworld" {
		t.Errorf("matching comparison modified the file: %q", got)
	}
}

func TestMismatchReportsFirstDifferingLine(t *testing.T) {
	inTempDir(t, "")
	path := filepath.Join("testdata", "x.golden")
	writeFile(t, path, "same\n\x1b[1mold\x1b[0m\nlast")
	makeReadOnly(t, path)

	stub := &stubTB{name: "TestX"}
	RequireEqual(stub, path, []byte("same\n\x1b[1mnew\x1b[0m\nlast"))

	if !stub.failed() {
		t.Fatal("expected a failure")
	}
	out := stub.output()
	if strings.Contains(out, "writing") {
		t.Errorf("mismatch tried to write the file:\n%s", out)
	}
	for _, want := range []string{
		path,
		"line 2",
		`"\x1b[1mold\x1b[0m"`,
		`"\x1b[1mnew\x1b[0m"`,
		"UPDATE_GOLDEN=1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("failure message missing %q:\n%s", want, out)
		}
	}
	if got := readFile(t, path); got != "same\n\x1b[1mold\x1b[0m\nlast" {
		t.Errorf("mismatch modified the file: %q", got)
	}
}

func TestMismatchTrailingNewline(t *testing.T) {
	inTempDir(t, "")
	path := filepath.Join("testdata", "x.golden")
	writeFile(t, path, "a\n")

	stub := &stubTB{name: "TestX"}
	RequireEqual(stub, path, []byte("a"))

	if !stub.failed() {
		t.Fatal("expected a failure")
	}
	if out := stub.output(); !strings.Contains(out, "line 2") || !strings.Contains(out, "<none>") {
		t.Errorf("message should point at the missing second line:\n%s", out)
	}
}

func TestMissingFileFailsAndCreatesNothing(t *testing.T) {
	dir := inTempDir(t, "")
	path := filepath.Join("testdata", "sub", "x.golden")

	stub := &stubTB{name: "TestX"}
	RequireEqual(stub, path, []byte("anything"))

	if !stub.failed() {
		t.Fatal("expected a failure")
	}
	out := stub.output()
	if !strings.Contains(out, path) || !strings.Contains(out, "UPDATE_GOLDEN=1") {
		t.Errorf("message should name the path and the hint:\n%s", out)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("compare mode created %d entries in the working directory", len(entries))
	}
}

func TestUpdateModeCreatesAndOverwrites(t *testing.T) {
	inTempDir(t, "1")
	path := filepath.Join("testdata", "deep", "dir", "x.golden")

	stub := &stubTB{name: "TestX"}
	RequireEqual(stub, path, []byte("first"))
	if stub.failed() {
		t.Fatalf("create failed: %s", stub.output())
	}
	if got := readFile(t, path); got != "first" {
		t.Fatalf("created file = %q, want %q", got, "first")
	}

	RequireEqual(stub, path, []byte("second"))
	if stub.failed() {
		t.Fatalf("overwrite failed: %s", stub.output())
	}
	if got := readFile(t, path); got != "second" {
		t.Fatalf("overwritten file = %q, want %q", got, "second")
	}

	// A following run without UPDATE_GOLDEN passes and changes nothing.
	t.Setenv("UPDATE_GOLDEN", "")
	makeReadOnly(t, path)
	RequireEqual(stub, path, []byte("second"))
	if stub.failed() {
		t.Fatalf("compare after update failed (a write attempt fails on a read-only file): %s", stub.output())
	}
	if got := readFile(t, path); got != "second" {
		t.Errorf("compare run modified the file: %q", got)
	}
}

func TestOnlyExactlyOneEnablesUpdateMode(t *testing.T) {
	for _, v := range []string{"true", "0", "yes", ""} {
		t.Run("UPDATE_GOLDEN="+v, func(t *testing.T) {
			dir := inTempDir(t, v)
			stub := &stubTB{name: "TestX"}
			RequireEqual(stub, filepath.Join("testdata", "x.golden"), []byte("data"))

			if !stub.failed() {
				t.Error("expected compare mode (missing file should fail)")
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Error("compare mode wrote files")
			}
		})
	}
}

func TestRequireViewPaths(t *testing.T) {
	tests := []struct {
		testName string
		w, h     int
		wantPath string
	}{
		{"TestPlaceholder", 80, 24, "testdata/TestPlaceholder/80x24.golden"},
		{"TestPlaceholder/tiny window", 10, 2, "testdata/TestPlaceholder_tiny_window/10x2.golden"},
		{"TestPlaceholder", 40, 10, "testdata/TestPlaceholder/40x10.golden"},
	}
	inTempDir(t, "1")
	for _, tt := range tests {
		stub := &stubTB{name: tt.testName}
		RequireView(stub, sizeModel{}, tt.w, tt.h)
		if stub.failed() {
			t.Fatalf("%s %dx%d: %s", tt.testName, tt.w, tt.h, stub.output())
		}
		want := fmt.Sprintf("size %dx%d\nsecond", tt.w, tt.h)
		if got := readFile(t, filepath.FromSlash(tt.wantPath)); got != want {
			t.Errorf("%s: content = %q, want %q (View must use the model returned by Update)", tt.wantPath, got, want)
		}
	}

	// Different sizes in one test use different files.
	if _, err := os.Stat(filepath.FromSlash("testdata/TestPlaceholder/80x24.golden")); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(filepath.FromSlash("testdata/TestPlaceholder/40x10.golden")); err != nil {
		t.Error(err)
	}

	// And compare mode passes against what update mode wrote.
	t.Setenv("UPDATE_GOLDEN", "")
	for _, tt := range tests {
		stub := &stubTB{name: tt.testName}
		RequireView(stub, sizeModel{}, tt.w, tt.h)
		if stub.failed() {
			t.Errorf("%s %dx%d compare failed: %s", tt.testName, tt.w, tt.h, stub.output())
		}
	}
}

func TestSanitize(t *testing.T) {
	tests := map[string]string{
		"TestX":             "TestX",
		"TestX/sub case":    "TestX_sub_case",
		"a.b_c-d":           "a.b_c-d",
		`Test/a\b:c*d?"e<>`: "Test_a_b_c_d__e__",
		"héllo":             "h_llo",
	}
	for in, want := range tests {
		if got := sanitize(in); got != want {
			t.Errorf("sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
