package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/jacob-lineberry/mhz/internal/app"
)

// version is set at build time with -ldflags "-X main.version=<value>".
// It must be a var, not a const, for -X to work.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. opts are extra Bubble Tea program options;
// production passes none, tests use them to drive the real program headlessly.
//
// Positional arguments are deliberately not validated yet: "mhz song.mp3" runs
// the TUI exactly like "mhz". A later change defines their behavior.
func run(args []string, stdout, stderr io.Writer, opts ...tea.ProgramOption) int {
	fs := flag.NewFlagSet("mhz", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(stderr, "Usage: mhz [--version]\n\nOptions:\n")
		fs.PrintDefaults()
	}
	showVersion := fs.Bool("version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		// flag has already printed the error and usage.
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *showVersion {
		_, _ = fmt.Fprintf(stdout, "mhz %s\n", version)
		return 0
	}

	if _, err := tea.NewProgram(app.New(), opts...).Run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "mhz: %v\n", err)
		return 1
	}
	return 0
}
