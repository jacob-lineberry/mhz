package main

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestRunFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		// substrings that must appear in stderr
		wantStderr []string
		// stderr must be empty
		quietStderr bool
	}{
		{
			name:        "version",
			args:        []string{"--version"},
			wantCode:    0,
			wantStdout:  "mhz dev\n",
			quietStderr: true,
		},
		{
			name:       "help",
			args:       []string{"--help"},
			wantCode:   0,
			wantStderr: []string{"-version", "--version"},
		},
		{
			name:       "short help",
			args:       []string{"-h"},
			wantCode:   0,
			wantStderr: []string{"-version"},
		},
		{
			name:       "unknown flag",
			args:       []string{"--bogus"},
			wantCode:   2,
			wantStderr: []string{"bogus", "Usage:", "--version"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr, tui bytes.Buffer
			// Same safety net as runHeadless: if a flag case ever fell
			// through to the TUI, it would quit on EOF or time out instead
			// of hanging on a real stdin.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			code := run(tt.args, &stdout, &stderr,
				tea.WithInput(strings.NewReader("")),
				tea.WithOutput(&tui),
				tea.WithContext(ctx),
			)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", code, tt.wantCode)
			}
			if stdout.String() != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.wantStdout)
			}
			if tt.quietStderr && stderr.Len() != 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
			for _, s := range tt.wantStderr {
				if !strings.Contains(stderr.String(), s) {
					t.Errorf("stderr = %q, want it to contain %q", stderr.String(), s)
				}
			}
			if tui.Len() != 0 {
				t.Errorf("TUI output = %q, want empty (TUI must not start)", tui.String())
			}
		})
	}
}

// runHeadless drives the real Bubble Tea program without a TTY. The 5 s
// timeout makes a broken quit path fail fast instead of hanging.
func runHeadless(t *testing.T, ctx context.Context, input string) (code int, stderr string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var errBuf bytes.Buffer
	code = run(nil, io.Discard, &errBuf,
		tea.WithInput(strings.NewReader(input)),
		tea.WithOutput(io.Discard),
		tea.WithWindowSize(80, 24),
		tea.WithContext(ctx),
	)
	return code, errBuf.String()
}

func TestRunQuitKeys(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"q", "q"},
		{"ctrl+c", "\x03"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stderr := runHeadless(t, context.Background(), tt.input)
			if code != 0 {
				t.Errorf("exit code = %d, want 0 (stderr: %q)", code, stderr)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code, stderr := runHeadless(t, ctx, "")
	if code == 0 {
		t.Error("exit code = 0, want non-zero")
	}
	if !strings.HasPrefix(stderr, "mhz:") {
		t.Errorf("stderr = %q, want prefix %q", stderr, "mhz:")
	}
}

func TestVersionInjectedWithLdflags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping build test in -short mode")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not on PATH")
	}

	name := "mhz"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(t.TempDir(), name)

	build := exec.CommandContext(t.Context(), goBin, "build", "-ldflags", "-X main.version=v1.2.3-test", "-o", bin, ".")
	build.WaitDelay = 10 * time.Second
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	versionCmd := exec.CommandContext(t.Context(), bin, "--version")
	versionCmd.WaitDelay = 10 * time.Second
	out, err := versionCmd.Output()
	if err != nil {
		t.Fatalf("running %s --version: %v", bin, err)
	}
	if got, want := string(out), "mhz v1.2.3-test\n"; got != want {
		t.Errorf("--version output = %q, want %q", got, want)
	}
}
