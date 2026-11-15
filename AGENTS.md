# AGENTS.md
 
mhz: a music-reactive terminal visualizer in Go (Bubble Tea v2, Lip Gloss v2, Bubbles v2, Harmonica).
 
## Commands
 
```bash
go build ./...
go test ./...
go test ./internal/<pkg> -run TestName   # single test  <!-- TODO: real path -->
go test -race ./...                      # required if you touched goroutines, channels, or audio
go vet ./...
golangci-lint run                        # config: .golangci.yml
gofmt -w <changed files>
go run .                                 # launch the TUI
```
 
## Definition of done
 
A task is done only when `gofmt`, `go build ./...`, `go vet ./...`, and `go test ./...` pass,
plus `go test -race ./...` if you touched concurrency. Report the commands you ran and their
results. Never claim validation passed unless you ran it and saw it pass.
 
If the same failure survives three fix attempts, stop and report the command, the error
output, and what you tried.
 
## Charm v2: most examples you know are v1
 
v1 code will not compile here. `go.mod` is the source of truth.
 
- Imports are `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, `charm.land/bubbles/v2/...`.
  Never `github.com/charmbracelet/{bubbletea,lipgloss,bubbles}`.
- `View()` returns `tea.View`, not `string`. Alt screen, mouse mode, window title, and cursor
  are fields on `tea.View`, not `tea.NewProgram` options or commands.
- Key input is `tea.KeyPressMsg` (fields `Code`, `Text`, `Mod`), not the v1 `tea.KeyMsg` struct.
  Space is `"space"`, not `" "`.
- Bubbles v2 components (e.g. viewport) use functional-option constructors and getters/setters
  instead of exported fields.
- Harmonica is `github.com/charmbracelet/harmonica` (v0.x). There is no v2.
- Unsure of an API? Read the source in `$(go env GOMODCACHE)` or the repo's
  `UPGRADE_GUIDE_V2.md`. Don't guess.
 
## Architecture rules
 
<!-- TODO: name the real packages once they exist. -->
 
- `View` only renders. State changes happen in `Update`; I/O and slow work run in `tea.Cmd`s
  and report back as messages.
- Nothing blocks the event loop. Long-running work (audio, streams) runs in a goroutine owned
  by one package, stops via `context.Context`, and talks to the program only through messages.
- Animation runs on a fixed tick (<!-- TODO: FPS -->). Create springs once with
  `harmonica.NewSpring(harmonica.FPS(n), ...)`; don't allocate per frame.
- No new dependencies without asking first.
 
## OpenSpec
 
Specs and tasks live in `openspec/`. Implement only the task you were given. If the spec is
ambiguous or wrong, stop and ask; don't invent requirements. Mark a task complete only when
the definition of done above is met.

When proposing a change:
- Read the relevant code and docs/decisions/ first.
- Ask me about anything ambiguous instead of deciding it.
- Size tasks so each one ends with `check` passing, tests included.
 
## Committing
 
ALWAYS use semantic commits (fix:, feat:, chore:, refactor:, docs:, sec:, etc).
Try to keep commits to one line. Only use multi-line commits when additional context is truly necessary.
 
## Don't
 
- Hand-edit `go.sum` (use `go get` / `go mod tidy`).
- Upgrade or downgrade dependencies to match an example.
- Weaken or delete tests to make them pass.
- Refactor unrelated code inside a feature change.
