# mhz roadmap

Researched and verified on 2026-11-16. OpenSpec owns the detail of each change; this file is
the order and the prompts. `<angle brackets>` mark decisions only you can make.

## 1. What mhz is

- A curated internet-radio player for the terminal that looks and feels like an 80s car
  stereo: your own stations on a made-up FM dial, each one a credited third-party stream.
- Decided: curated only (embedded list plus user stations in config, no search UI). Native
  Go decode and playback, MP3 first. Your own station names and text idents, broadcaster
  credited on screen, no GTA branding.
- Order: **M0** clear the deck (chore 0.2; chore 0.3 any time) → spike S1 →
  **M1** radio core (spike S2 before change 2) → **M2** faceplate → **M3** first release
  (v0.1.0) → **M4** spectrum analyzer → **M5** full-screen scenes (the rain scene is built
  fresh in change 11) → **M6** reach (more codecs, more package channels).

### Where the repo differs from the brief

| Brief | Repo / GitHub on 2026-11-16 | Effect |
|---|---|---|
| OpenSpec 1.x, `/opsx:*` | CLI 2.0.0. Skills: `/openspec-explore`, `-propose`, `-apply-change`, `-update-change`, `-sync-specs`, `-verify-change`, `-archive-change` | Prompts use these names |
| First commit on a branch is `openspec/changes/<name>/` | `openspec/`, `.claude/` and `.agents/` are in `.gitignore` | They stay ignored. Specs, changes and archives live only on this machine; PRs carry code and docs. Copy `openspec/` to the MacBook yourself |
| Rain scene exists on a branch | An earlier `add-rain-scene` branch (20/26 tasks) was never merged and was deleted on 2026-11-16 | Nothing to merge. Change 11 builds the rain scene fresh; its look values are in that prompt |
| Checks required and up to date | `strict_required_status_checks_policy` is `false` | §7 |
| CodeQL and secret scanning to add | Already on: CodeQL default setup (go, actions; weekly), secret scanning, push protection, Dependabot alerts, read-only default token. Off: private vulnerability reporting, SHA-pin requirement (Dependabot security updates were switched on 2026-11-16) | §7 is shorter |
| Bootstrap had 26 tasks | Bootstrap had 19; the rain change has 26 | Same lesson |
| — | README and AGENTS.md line 1 still say "music-reactive terminal visualizer"; AGENTS.md has two TODOs; `docs/decisions/` doesn't exist | Chore 0.2 |

Current versions (module proxy): bubbletea v2.1.1, lipgloss v2.1.1 (2026-11-15), bubbles
v2.2.1, harmonica v0.2.0, golangci-lint v2.15.0 (your pin is the latest).

### Research findings that shape the plan

- **No cgo needed.** [oto] v3.5.1 lists Linux, macOS and Windows as "no Cgo required". On
  Linux it talks to PulseAudio in pure Go and falls back to loading `libasound.so.2` at
  runtime. A `CGO_ENABLED=0` binary opened a device on this WSL2 machine through WSLg, and
  cross-builds succeeded for all six targets. Releases, CI and CodeQL stay simple.
- **The usual MP3 decoder is archived.** [go-mp3] v0.3.4 (2022) is read-only and rejects
  MPEG 2.5 and free-format streams. Candidates: [tphakala/go-mp3] v0.4.0 (days old, 1 star)
  and [llehouerou/go-mp3] v1.5.0 (fork). S1 decides.
- **oto allows one context per process at one sample rate.** A station at another rate
  needs resampling or exclusion. S1 decides.
- **Nightride FM is a good anchor technically.** Seven listed stations, all
  `https://stream.nightride.fm/<slug>.mp3`, MP3 320 kbps, `icy-metaint: 16000` (curl,
  2026-11-16). `icy-name` is an opaque id, so names must come from your list. They publish
  no policy on third-party players.
- **SomaFM is out.** Its [terms][somafm-tos] (updated 2026-09-04) say "We can't grant
  permission for third-party SomaFM clients or applications, even noncommercial ones."
- **AAC is the hard codec.** Pure-Go decoders are AAC-LC only and LGPL/GPL; none handles
  HE-AAC. Opus is closer: [pion/opus] v0.1.0 decodes SILK and CELT. Both wait for M6.
- **Prior art:** [cliamp] (Bubble Tea v2, beep, oto, PCM tap, own FFT, ICY shim). It shells
  out to ffmpeg for AAC and Opus. Read it before S1.
- **GoReleaser's Homebrew path changed.** `brews` is [deprecated][gr-deprecations];
  `homebrew_casks` ships prebuilt binaries and the docs say casks "are supposed to be
  signed". Plan: a hand-written source-build formula, no GoReleaser Homebrew step.

## 2. Workflow per change

| Step | What | Where |
|---|---|---|
| 1. Branch | `git switch main && git pull && git switch -c feat/<name>`, only after the previous change is merged | shell |
| 2. Explore (when marked) | `/openspec-explore` with the change's intent; no files written. Fold the answers into the prompt's placeholders | new session |
| 3. Propose | `/openspec-propose` plus the prompt from §5 | new session |
| 4. Your review | Read proposal → specs → tasks, design last. Send back if: more than 12 tasks, tooling in specs, implementation numbers in specs, a missing out-of-scope list. Fix with `/openspec-update-change` | same session |
| 5. Apply | `/openspec-apply-change <name>`, then "do group N only, then stop". One group per session, `/clear` between groups, one commit per group | new session per group |
| 6. Independent review | Review prompt below, different model from the one that applied. Then `/openspec-verify-change <name>` | new session |
| 7. Hands-on | Run it. Look at stills from `cmd/frame`. Listen. Use the change's done-when list | you |
| 8. Archive | `/openspec-archive-change <name>`, before merging. Local only; nothing to commit | same branch |
| 9. Merge | Push, open the PR with a semantic title, squash-merge, delete the branch | GitHub |

Tooling work (CI, release, packaging) skips steps 2–4, 6 (verify) and 8: it is a plain
chore PR with a prompt from §5, because specs describe product behavior only.

### Standing rules (goes in `openspec/config.yaml` in chore 0.2)

This keeps the lessons out of every prompt. The file is gitignored, so it lives on each
machine you work on. If it's missing, paste the `rules` text under each propose prompt.

```yaml
context: |
  mhz is a curated internet-radio player for the terminal with an 80s car-stereo
  faceplate. Read docs/decisions/ before proposing; for visual changes also read
  docs/design/.
  Agents cannot see the terminal or hear audio. Every visual requirement needs a
  golden file or a cmd/frame still. Every audio requirement needs a test against a
  fake output device.
rules:
  proposal:
    - List what is out of scope.
    - Name every new dependency by module path. Only those approved in the prompt
      may be added.
  specs:
    - Describe behavior a user can observe. No tooling, package layout, CI, or
      implementation numbers such as buffer sizes, FFT sizes or per-row rules.
    - Give timing requirements as a bound a test can check with a fake clock.
  design:
    - Where goldens depend on arithmetic, use integer or fixed-point math, or
      quantize before rendering. Seed math/rand/v2. Code under test never reads
      the wall clock.
    - Program-loop tests pin tea.WithColorProfile, pre-size the model, and use
      WithInput plus a context timeout.
    - Tests never open a real audio device or the network.
  tasks:
    - 6 to 12 tasks in 2 to 4 groups. Each group is one session and one commit.
    - Visual work has a task where I approve stills before any motion task.
    - If the spec and the design conflict, stop and ask.
```

### Review prompt (reusable)

```text
Review change <name> on branch <branch>. You did not write it. Do not edit, commit
or push anything in this worktree.

Read openspec/changes/<name>/ (proposal, specs, design, tasks), then
`git diff main...HEAD`.

1. Spec coverage. For every scenario in the delta specs, name the test that proves
   it, or write "untested".
2. Mutation check. Create a throwaway worktree: `git worktree add /tmp/mhz-review HEAD`.
   Pick the <5> behaviors most likely to regress. Break each one there, one at a
   time (flip a condition, drop a call, change a constant), and run the narrowest
   test. Record "mutant → caught by <test>" or "survived". Remove the worktree at
   the end.
3. Concurrency and lifecycle. Check goroutine ownership, context cancellation,
   blocking sends, and anything that could hang quit. Run
   `go test -race -count=20` on packages that start goroutines.
4. Scope. List anything in the diff the tasks did not ask for, and any spec text
   that describes tooling or implementation detail.

Report findings ranked Blocker / Should fix / Nit. Each finding has file:line,
what fails, the evidence (command and output), and the smallest fix. Then list
the surviving mutants. Fix nothing. Finish with what you did not check.
```

### Spike prompt template

```text
Spike: <question>. This is throwaway research, not a change. Write no OpenSpec
artifacts.

Work on branch spike/<name>, inside spikes/<name>/ with its own go.mod, so the
main module and go.sum stay untouched. Only the ADR merges to main.

Answer these questions with evidence (commands, numbers, logs), in order:
<numbered questions>

Limits: <platforms>; one session. I run anything that needs ears or a second
machine: give me the exact command and what to listen for, then wait.

Deliver a draft docs/decisions/<NNNN>-<slug>.md with: Context, Options considered
(with measurements), Recommendation, Consequences, Open questions for me, and how
to reproduce each measurement. Mark every claim as measured, read in source, or
assumed. Stop after the draft. I make the decision.
```

## 3. Model and effort plan

Facts this rests on ([model config][cc-model], [prompt caching][cc-cache],
[pricing][claude-pricing], [fast mode][cc-fast]):

- Claude Code defaults every current model to `medium` effort. Set it with `/effort high`
  or `/effort xhigh`; the choice is saved per model.
- Switching models mid-session discards the prompt cache. Changing effort on Opus 5.5 or
  Sonnet 5.5 does not. Pick the model when the session starts.
- Max 5x has a five-hour limit and a weekly limit. Fable has its own weekly limit,
  "up to half your weekly limit". Exact numbers aren't published.
- Fast mode is billed to usage credits, outside the plan. Leave it off.
- `opusplan` switches model on every plan-mode toggle, which discards the cache each time.
  One model per session is cheaper.

| Step | Model | Effort | Why |
|---|---|---|---|
| Spike | Opus 5.5 | High | Needs judgment about evidence, not polish; code is thrown away |
| Explore | Opus 5.5 | High | Conversation only; depth comes from your answers |
| Propose: audio and concurrency (changes 1, 2, 9) | Fable 5.2 | High | A wrong design here costs the most rework, and a proposal is one short session. Fable's gains above High are uneven |
| Propose: visual or determinism-heavy (6, 8, 10, 13, 14) | Opus 5.5 | XHigh | Many easy-to-miss cases (size tiers, color fallback, golden stability) |
| Propose: everything else | Opus 5.5 | High | Small, well-bounded slices |
| Apply: ordinary task groups | Sonnet 5.5 | High | Tasks are already specified; this is where most tokens go |
| Apply: goroutine, audio-thread or reconnect groups | Opus 5.5 | High | Races and lifecycle bugs are cheap to write and slow to find |
| Independent review | Opus 5.5 | XHigh | Fresh context and a different model from the applier; the mutation check needs care |
| Independent review of changes 1, 2, 9 | Fable 5.2 | High | Concurrency bugs pass tests; this is the last look before your ears |
| Still-tuning loops | Sonnet 5.5 | High | Many small iterations and you are the judge |
| Verify, archive, chores, CI, release | Sonnet 5.5 | High | Mechanical |

**Where Fable is worth it:** three proposals and three reviews, about six short sessions
across the project. **Where it isn't:** apply sessions (long tool loops would spend the
Fable share fastest), still tuning, chores, archive. If the Fable limit runs out, use Opus
5.5 at XHigh for the same steps; nothing else changes.

## 4. Change list

"Chore" rows are plain PRs without OpenSpec. The last column is the model and effort for
the propose step (or for the whole chore).

| # | Milestone | Change | Depends on | Explore first? | Propose |
|---|---|---|---|---|---|
| 0.2 | M0 | chore `pivot-housekeeping` | — | no | Sonnet / High |
| 0.3 | M0 | chore `ci-hardening` | — (any time) | no | Sonnet / High |
| S1 | spike | `audio-path` → ADR-0001 | 0.2 | — | Opus / High |
| S2 | spike | `stream-resilience` → ADR-0002 | S1 | — | Opus / High |
| 1 | M1 | `add-stream-playback` | ADR-0001 | yes | Fable / High |
| 2 | M1 | `add-stream-resilience` | 1, ADR-0002 | yes | Fable / High |
| 3 | M1 | `add-now-playing` | 1 | no | Opus / High |
| 4 | M1 | `add-station-catalog` | 1 | no | Opus / High |
| 5 | M1 | `add-volume-and-mute` | 1 | no | Opus / High |
| 6 | M2 | `add-faceplate` | 3, 4, 5 | yes | Opus / XHigh |
| 7 | M2 | `add-presets-and-help` | 6 | no | Opus / High |
| 8 | M2 | `add-faceplate-motion` | 6 | yes | Opus / XHigh |
| R1 | M3 | chore `release-pipeline` | 7 | no | Sonnet / High |
| R2 | M3 | chore `install-and-tap` | R1 | no | Sonnet / High |
| S3 | spike | `audio-analysis` → ADR-0003 | 1 | — | Opus / High |
| 9 | M4 | `add-audio-levels` | 6, ADR-0003 | yes | Fable / High |
| 10 | M4 | `add-spectrum-analyzer` | 9 | yes | Opus / XHigh |
| 11 | M5 | `add-scene-mode` | 9 | no | Opus / High |
| 12 | M5 | `add-rain-reactivity` | 11 | yes | Opus / High |
| 13 | M5 | `add-<scene>-scene` (repeatable) | 11 | yes | Opus / XHigh |
| S4 | spike | `more-codecs` → ADR-0004 | 2 | — | Opus / High |
| 14 | M6 | `add-<codec>-streams` | ADR-0004 | yes | Opus / XHigh |
| R3 | M6 | chore `package-channels` | R1 | no | Sonnet / High |

## 5. Prompts

Paste each block after `/openspec-propose` unless it says otherwise.

### M0: clear the deck

**0.2 Chore `pivot-housekeeping`** (first; branch `chore/pivot-housekeeping` from `main`,
no OpenSpec):

```text
Housekeeping after the pivot to a radio player. No product behavior changes.

1. openspec/config.yaml: add the context and rules block from docs/roadmap.md §2.
   The file is gitignored, so it won't appear in the diff. Run
   `openspec validate --specs --strict` and fix nothing but the config.
2. README.md: one-line description of the radio player, a status line ("pre-release"),
   and a link to docs/roadmap.md.
3. AGENTS.md: fix line 1 to match; fill the FPS TODO (decided: 30); replace the
   single-test TODO with a real example path; add "spikes/ is throwaway and has its
   own go.mod" and "openspec/, .claude/ and .agents/ are gitignored; never stage
   them". Keep it short.
4. docs/decisions/README.md: one paragraph on what an ADR is here, plus a template
   (Status, Context, Options, Decision, Consequences).

Done when: `make check` passes and the diff touches only README.md, AGENTS.md and
docs/decisions/README.md.
```

**0.3 Chore `ci-hardening`** (branch `chore/ci-hardening`): prompt in §7.

### M1: radio core

**1. `add-stream-playback`**

```text
Change: add-stream-playback

Intent: launching mhz plays one built-in station through the default audio device.
I can stop and resume it, and the screen tells me what is happening.

Decisions already made:
- Audio stack per docs/decisions/0001-audio-path.md. Approved new dependencies:
  github.com/ebitengine/oto/v3 and <the MP3 decoder ADR-0001 picked>. No cgo.
- One built-in station for now: https://stream.nightride.fm/nightride.mp3.
- Requests send User-Agent "mhz/<version> (+https://github.com/jacob-lineberry/mhz)".
- Until the faceplate exists, the TUI is a plain text status screen. It replaces
  the bootstrap placeholder.
- Tests use a fake output device and a local test server that serves a short
  test-tone MP3 from testdata (I will supply or approve the file).
- Autoplay on launch: yes (the built-in station until change 4 remembers the last
  one).

In scope: connect, decode, play; states stopped / connecting / playing / error
shown on screen; space toggles play and stop; quit silences audio and exits
promptly; a missing audio device or failed connection shows an error state and
the TUI keeps running.

Out of scope: more than one station, reconnecting after a drop (a dropped stream
ends in the error state), track titles, volume, the faceplate, any visualization.

Done when:
- `go run .` on WSL2 plays the station within <5> s and space stops and resumes it.
- q exits within <1> s with no audio left playing and no process left behind.
- With the network off, the error state appears within <10> s and nothing hangs.
- `make check` passes on all three CI systems, none of which has an audio device.
```

**2. `add-stream-resilience`**

```text
Change: add-stream-resilience

Intent: a flaky network never needs a restart. The player reconnects by itself
and shows what it is doing.

Decisions already made:
- Timeouts, backoff schedule and buffer targets per
  docs/decisions/0002-stream-resilience.md.
- Be a polite client: never retry faster than <1> s apart, back off exponentially
  with jitter up to <30> s, one connection at a time.
- A user stop cancels any pending retry at once.
- Time is injected, so tests use a fake clock and never sleep.

In scope: stall detection; reconnect after EOF, reset or timeout; states
buffering and reconnecting (with attempt count and time to next retry) on
screen; permanent failures (404, wrong content type, undecodable stream) shown
as failed without retrying; HTTP redirects.

Out of scope: fallback URLs per station, playlists (.pls, .m3u), HLS, legacy
"ICY 200 OK" servers <unless a station in my list needs it>.

Done when:
- Tests against a test server that drops, stalls and returns 404 prove the exact
  sequence of states for each case.
- Hands-on: I cut the network for 20 s and playback resumes without input.
- Stop during a reconnect takes effect immediately.
- `go test -race` is clean.
```

**3. `add-now-playing`**

```text
Change: add-now-playing

Intent: I can see what track is playing.

Decisions already made:
- Source is in-band ICY metadata (request header Icy-MetaData: 1, icy-metaint,
  StreamTitle), parsed in-house with no new dependency.
- Show the title as received. Strip control characters and escape sequences so a
  stream can't write to my terminal. Treat invalid UTF-8 as Latin-1.
- No broadcaster-specific metadata APIs.

In scope: metadata is removed from the audio before decoding; the screen updates
when the title changes; stations with no metadata show the station only; long
titles are cut with an ellipsis to fit.

Out of scope: history, scrobbling, artwork, marquee scrolling.

Done when:
- Byte-fixture tests cover: a metadata block split across reads, empty blocks, a
  bad length byte, non-UTF-8 text, and escape sequences.
- Decoded audio for the fixture is byte-identical with metadata on and off.
- Hands-on: the title changes when the track changes on the built-in station.
```

**4. `add-station-catalog`**

```text
Change: add-station-catalog

Intent: several stations under my own names on a made-up FM dial. I can tune
between them and add my own in a config file.

Decisions already made:
- Stations are an embedded list plus a user file in the OS config directory.
  Format: TOML. Approved new dependency: github.com/BurntSushi/toml.
- Each station has: id, my name for it, FM frequency (87.5–108.0, unique), a
  short ident line, genre, stream URL, broadcaster name, broadcaster homepage.
- MP3 streams only. The broadcaster credit is always visible while tuned.
- Embedded stations (name / frequency / ident / stream):
  <fill in from the verified table in docs/roadmap.md §6>
- The last tuned station is remembered between runs and plays on launch.
- `mhz --stations` prints the catalog and exits.

In scope: tune up and down in frequency order with wraparound; switching stops
the old stream before the new one is heard; user stations add to or override the
embedded ones by id; a broken user file gives an error naming the file and the
problem while embedded stations still work.

Out of scope: search, editing stations inside the TUI, favorites, importing
playlists, non-MP3 streams.

Done when:
- Tuning through every embedded station plays each one, with no overlap.
- A user file adds a station and overrides one; a broken file is reported clearly.
- Restarting returns to the last station and plays it.
- `mhz --stations` output is covered by a golden test.
```

**5. `add-volume-and-mute`**

```text
Change: add-volume-and-mute

Intent: volume and mute from the keyboard, remembered between runs.

Decisions already made:
- Gain is applied in software to the decoded audio. Any later analysis reads the
  audio before gain, so visuals don't shrink when I turn it down.
- Up and down change volume in steps of <5>; m toggles mute; default <70>.
- The curve is perceptual, not linear: <yes | no>.

In scope: volume and mute state on screen; no clipping at full volume; mute keeps
the stream connected; both persist with the last station.

Out of scope: system mixer control, per-station volume, fades.

Done when:
- Fake-device tests show sample values scale as specified and never exceed full
  scale.
- Hands-on: steps sound even across the range.
- Volume and mute survive a restart.
```

### M2: faceplate

**6. `add-faceplate`** (explore first: layout options as stills)

```text
Change: add-faceplate

Intent: replace the plain status screen with the car-stereo faceplate.

Decisions already made:
- docs/design/faceplate.md is the visual brief: palette roles, hierarchy, size
  tiers, glyph rules. Follow it; ask before departing from it.
- Approved new dependency: charm.land/lipgloss/v2.
- The faceplate renders from a plain view-state value, so stills need no audio.
- cmd/frame gains a way to print the faceplate for named fixture states (playing,
  connecting, reconnecting, failed, muted, long title, no metadata).
- No animation in this change. Every state is a static picture.
- The analyzer window is laid out now and shows unlit segments.

In scope: frequency in large digits, station name, ident, broadcaster credit,
state and stream info, dial with needle, now-playing line, volume bar, key hints,
the three size tiers and the two small-terminal fallbacks.

Out of scope: presets, help overlay, motion, spectrum, themes, light mode, mouse.

Done when:
- I have approved stills at 80x24, 120x40 and 240x70 and both fallbacks, before
  the app is wired to them.
- Every size from 1x1 to 240x70 renders exactly that size without panicking.
- Stills forced to 256 colors, 16 colors and no color still show the station,
  the state and the volume.
- `go run .` matches the stills for each state I can reach by hand.
```

**7. `add-presets-and-help`**

```text
Change: add-presets-and-help

Intent: six preset buttons like a car stereo, and keys I can discover.

Decisions already made:
- Keys 1–6 recall a preset. s then 1–6 stores the current station in that slot.
- Default presets: <the first six embedded stations | my picks>.
- Approved new dependency: charm.land/bubbles/v2 (help and key bindings only).
- The footer shows short key hints; ? opens full help with a credits section
  listing each broadcaster, its URL, and a line saying mhz is not affiliated.

In scope: preset row on the faceplate with the active slot marked; presets
persist; help overlay closes with ? or esc; hints shrink gracefully when narrow.

Out of scope: more than six presets, bands (FM1/FM2), mouse clicks.

Done when:
- Stills of the preset row and help overlay are approved at 80x24 and 120x40.
- Stored presets survive a restart.
- Every key binding in the app appears in the full help (tested).
```

**8. `add-faceplate-motion`** (explore first: which motions earn their place)

```text
Change: add-faceplate-motion

Intent: the faceplate feels alive without becoming noisy.

Decisions already made:
- Motion runs on the existing fixed frame tick.
- Approved new dependency: github.com/charmbracelet/harmonica. Springs are
  created once.
- Motions: the needle glides to the new frequency on tune; titles longer than
  their field scroll, pausing at each end; connecting and reconnecting pulse.
  Digits change instantly: <yes | roll>.
- When nothing is animating, the screen does not redraw: <yes | always tick>.
- Reduced-motion switch: <flag name | none>.
- Goldens must be stable on arm64 and amd64. If spring math makes that hard,
  stop and ask before working around it.

In scope: the three motions above and their settle behavior.

Out of scope: spectrum, scenes, color cycling, transitions between screens.

Done when:
- cmd/frame stills at chosen times show each motion mid-way and settled, and I
  have approved them.
- The needle settles within <400> ms of a tune.
- Idle CPU with the faceplate open is under <2>% on my machine.
```

### M3: first release

Prompts for R1 and R2 are in §8.

### M4: spectrum analyzer

**9. `add-audio-levels`** (explore first)

```text
Change: add-audio-levels

Intent: the app can show what it hears. A stereo level meter on the faceplate
proves the audio tap end to end.

Decisions already made:
- Tap position, delay compensation and analysis window per
  docs/decisions/0003-audio-analysis.md.
- Analysis never blocks or slows playback. If the UI falls behind, analysis
  frames are dropped.
- Levels reach the UI as messages on the frame tick, through scene.Features.
  Its zero value still means "no audio".
- The tap reads audio before volume and mute are applied.

In scope: left and right level meters with fast attack and slow decay; levels
fall to zero when stopped or buffering.

Out of scope: frequency bands, scenes reacting to audio.

Done when:
- Synthetic audio (silence, full-scale tone, left-only) produces exact, golden
  meter cells.
- A test with a stalled UI consumer shows playback output is unaffected.
- Hands-on: ten minutes of listening with no glitches; the meter tracks the music.
- `go test -race -count=20` is clean on the audio packages.
```

**10. `add-spectrum-analyzer`** (explore first: band count and ballistics as stills)

```text
Change: add-spectrum-analyzer

Intent: the 80s spectrum analyzer inside the faceplate.

Decisions already made:
- FFT approach per ADR-0003. Approved new dependency:
  <gonum.org/v1/gonum (dsp/fourier) | none, in-house>.
- Bands are log-spaced; the count follows the size tier (<18> at 80 columns).
- Bars rise fast and fall slowly. Each has a peak cap that holds for about
  <1> s, then drops.
- Requirements describe what I see (a 1 kHz tone lights the band containing
  1 kHz the most), not how it is computed.
- Bar heights must not differ between arm64 and amd64 for the test signals.

In scope: bars and peak caps in the analyzer window at every size tier; silence
shows unlit segments.

Out of scope: analyzer styles, user-set band counts, scenes.

Done when:
- Golden stills for silence, single tones, a sweep and pink noise are approved.
- The same goldens pass on all three CI systems.
- Playing audio costs under <5>% CPU at 120x40 on my machine.
```

### M5: full-screen scenes

**11. `add-scene-mode`**

```text
Change: add-scene-mode

Intent: one key swaps the faceplate for a full-screen rain scene and back, while
the radio keeps playing.

Decisions already made:
- Key: <v>.
- The rain scene is built fresh here: a pixel framebuffer, a half-block encoder (upper
  half block, foreground = top pixel, background = bottom pixel), a scene contract
  (Resize, Update, pure Draw) and a cmd/frame still tool. The look is below.
- Look (a half-block pixel is about square; speeds are absolute pixels per second, so a
  bigger window shows more sky):
  - Sky: 9 flat bands plus a one-row horizon glow, dark-heavy so most of the frame
    reads as night. Steps `#07061A #0C0A28 #140E38 #1F1149 #2E1458 #431763 #5E1A6B
    #82206F #B02A74`, glow `#FF4F9A`. Band starts in 64ths of sky height: 0, 16, 27,
    36, 43, 49, 54, 58, 61. The last 3 rows of each band blend toward the next with
    a 2x2 ordered dither, and the brighter step's share never falls going down.
  - Ground: 4 px tall (H/4 under 16 rows). Top row `#040308`, rest `#020204`, both
    darker than the darkest sky step.
  - Rain: three layers drawn far to near. Density in drops per px^2, length in px,
    speed in px/s (each drop varies +-10%): far 1/60, 2, 45; mid 1/120, 4, 75;
    near 1/240, 8, 120. Far and mid drops are tinted toward `#D8C8FF`
    (bg + (`#D8C8FF` - bg) * k / 256, k = 80 and 150, per channel) so they are
    always lighter than the sky behind them. Near drops are `#D9C2FF` with a
    `#FFFFFF` head pixel.
  - Wind: 1 px across per 4 px down. A streak's shape depends only on the pixel
    index along it, so it never wobbles as it moves.
  - Splashes: on the landing row itself, 3 fade steps of 80 ms, gone by 240 ms.
    Near `#FFFFFF #B8A0F0 #5A4A90`, mid `#C8B4FF #7A62C0 #3A2E66`, far
    `#8A78C8 #4A3E7A #2A2248`.
  - These values fixed four faults of an earlier first pass: a magenta checkerboard
    sky, drops darker than the sky, wobbling streaks and splashes floating above the
    ground. Tune them from stills, not by eye in the terminal.
- Determinism: integer fixed-point motion, a seeded PCG, a pure Draw, and the
  frame time passed in. Goldens are stable on arm64 and amd64.
- In scene mode a one-line strip shows frequency, station and title. It
  <hides after 5 s and returns on any key | stays>.
- Tuning, presets, volume, mute and play/stop work in scene mode.
- The scene receives scene.Features every frame; its zero value means "no audio".

In scope: the pixel buffer, encoder and scene contract; the rain scene; cmd/frame
stills for it; the toggle; the strip; remembering the last mode between runs.

Out of scope: choosing between scenes, transitions, the scene reacting to audio.

Done when:
- Stills of the rain at t=0 and two later times at 80x24, 120x40 and 240x70 are
  approved before any motion tuning.
- Stills of scene mode with and without the strip are approved.
- Toggling never interrupts audio (fake-device test).
- Sizes from 0x0 upward never panic; the 240x70 frame benchmark stays under 1 ms.
```

**12. `add-rain-reactivity`** (explore first: what should respond to what)

```text
Change: add-rain-reactivity

Intent: the rain responds to the music.

Decisions already made:
- Mapping: <e.g. overall level → rain density, bass → splash brightness>.
- With zero-value Features the scene looks exactly as it does today.

In scope: the mapping above, smoothed so it never flickers.

Out of scope: new scene elements (city, lightning), other scenes.

Done when:
- Change 11's rain goldens are byte-identical with zero-value Features.
- Stills for quiet, loud and bass-heavy synthetic input are approved.
- The 240x70 frame benchmark stays under 1 ms.
```

**13. `add-<scene>-scene`** (repeatable; explore first)

```text
Change: add-<scene>-scene

Intent: a new full-screen scene: <one-sentence description of the picture and mood>.

Decisions already made:
- Palette: <rain palette | faceplate palette | new, listed here>.
- Reacts to audio: <mapping | static only>.
- It follows the existing scene contract and determinism rules.
- Scene selection: <a key cycles scenes | config only>.

In scope: <elements, back to front>.
Out of scope: <elements deferred>.

Done when:
- Stills at t=0 and two later times at 80x24 and 120x40 are approved before any
  motion tuning.
- Sizes from 0x0 upward never panic; the frame benchmark stays under 1 ms.
```

### M6: reach

**14. `add-<codec>-streams`** (only if ADR-0004 says go)

```text
Change: add-<codec>-streams

Intent: stations that only offer <codec> can be in the catalog.

Decisions already made:
- Decoder and license position per docs/decisions/0004-more-codecs.md. Approved
  new dependency: <module>.
- The catalog marks each station's codec; MP3 behavior is unchanged.

In scope: decode and play <codec> streams; now-playing for them if the container
carries titles; a clear "unsupported stream" state for anything else.

Out of scope: HLS, DRM streams, transcoding.

Done when:
- A test stream fixture decodes to the expected audio in a fake-device test.
- Hands-on: <station> plays for 30 minutes without glitches.
- All existing goldens and MP3 tests are unchanged.
```

R3 prompt is in §8.

## 6. Spikes

Each spike uses the template in §2 and ends in a draft ADR that you finalize.

| Spike | Before | ADR | Questions |
|---|---|---|---|
| S1 `audio-path` | change 1 | `0001-audio-path.md` | See below |
| S2 `stream-resilience` | change 2 | `0002-stream-resilience.md` | See below |
| S3 `audio-analysis` | change 9 | `0003-audio-analysis.md` | See below |
| S4 `more-codecs` | change 14 | `0004-more-codecs.md` | See below |

**S1 questions** (platforms: WSL2 now, native Windows 11 now, macOS when the MacBook
arrives; the ADR stays draft until all three are checked):

1. Which MP3 decoder: archived [go-mp3], [tphakala/go-mp3], or [llehouerou/go-mp3]? Compare
   on a 30-minute 320 kbps live stream: CPU, glitches, behavior when joining mid-frame,
   behavior on corrupt bytes, API fit for a non-seekable reader.
2. Does oto v3.5.1 play cleanly cgo-free on each platform? What buffer size gives no
   underruns, and what latency does that imply?
3. Over 30 minutes, does the buffer between a live stream and the sound card grow or drain
   (clock drift)? What policy handles it?
4. oto fixes one sample rate per process. What do the candidate stations use, and is the
   answer "44.1 kHz only", or resampling?
5. Does in-band ICY parsing work against Nightride and Nightwave Plaza? Does any candidate
   station answer `ICY 200 OK`, which Go's HTTP client rejects?
6. What does a test double for the output device look like, so CI never needs audio?

**S2 questions:**

1. What actually happens to the S1 pipeline when the network drops, the server stalls, or
   the server closes the stream? Measure time to detect each.
2. Which stall timeout and backoff schedule recover fastest without hammering a
   broadcaster? Propose numbers.
3. How much audio should be buffered before playback starts and after a reconnect, and what
   does the listener hear at each choice?
4. Can a local test server reproduce every failure deterministically with a fake clock?

**S3 questions:**

1. Where should the tap sit so the picture matches what is heard? Measure the delay between
   tap and speaker on each platform.
2. FFT cost at 1024–4096 samples at the frame rate. (Measured here: gonum real FFT takes
   7.5 µs at 1024 with no allocations.) Is gonum worth a dependency over an in-house FFT?
3. How do float results differ between arm64 and amd64, and what quantization keeps bar
   heights identical?
4. Which band layout and attack/decay values look right? Produce stills from recorded audio.

**S4 questions:**

1. Does [pion/opus] plus an Ogg reader play Nightride's `.ogg` (Opus) stream cleanly?
2. Is there a pure-Go HE-AAC decoder yet? If only GPL or LGPL options exist, what does that
   mean for an MIT project?
3. Which stations you want are blocked on each codec?

**Decision without a spike, `0005-stations-and-credits.md` (yours):** which stations ship,
your names and frequencies, credit wording, and what each broadcaster said when asked.

### Verified candidate streams (curl, 2026-11-16)

| Broadcaster | Stream | Codec | ICY titles | Policy on third-party players |
|---|---|---|---|---|
| [Nightride FM][nightride]: nightride, chillsynth, datawave, spacesynth, darksynth, horrorsynth, ebsm | `https://stream.nightride.fm/<slug>.mp3` | MP3 320k | yes | None published. Ask on their Discord before v0.1.0 |
| [Nightwave Plaza][plaza] | `http://radio.plaza.one/mp3` | MP3 128k | yes | None published; the site offers `.m3u`/`.pls` for external players |
| Fantasy Radio (italo) | `https://italo.live-streams.nl/fantasy` | MP3 256k | yes | Not found |
| laut.fm Synthwave | `http://stream.laut.fm/synthwave` (redirects) | MP3 128k | yes | Not found |
| Frekvenci Night Drive | `https://stream.frekvenci.com/nightdrive` | MP3 128k | yes | Has terms; unread |
| SomaFM | — | — | — | **Refused** by their [terms][somafm-tos]; do not embed |

## 7. CI/CD

| When | Add | Settings and ruleset changes |
|---|---|---|
| Now (chore 0.3, trimmed; done in d4e7563) | `.github/dependabot.yml`; actions pinned to commit SHAs; a `govulncheck` workflow (push, PR, weekly) | Dependabot security updates and auto-delete of merged branches: on |
| 2026-11-21 (done) | a `zizmor` workflow (push, PR, weekly) that audits `.github/`; `persist-credentials: false` on the `ci.yml` checkout | Require SHA-pinned actions: on. Squash commits use the PR title with a blank body. Add `zizmor` to the required checks |
| Before v0.1.0 (optional) | `.github/SECURITY.md`; one aggregate `check` job | Enable private vulnerability reporting. Require only `check` (and `zizmor`). "Require branches to be up to date". "Require code scanning results" for CodeQL |
| With change 1 | Nothing if ADR-0001 stays cgo-free | If a cgo dependency is chosen: CodeQL default setup can't install system packages, so switch to [advanced setup][codeql-build] with `build-mode: manual`, and install the headers in `ci.yml` too |
| With R1 | Release workflow; `goreleaser check` in CI | Tag ruleset for `v*` (no update, no delete). Enable immutable releases. The release job alone gets `contents: write`, `id-token: write`, `attestations: write` |
| Later, if PR volume annoys you | Dependabot auto-merge for patch updates ([pattern][dependabot-automerge]) | Allow auto-merge |

Notes:

- **Already on, nothing to do:** CodeQL default setup, secret scanning, push protection,
  Dependabot alerts, read-only default `GITHUB_TOKEN`.
- **Trimmed on 2026-11-16.** Modeled on Crush's workflows: it has Dependabot, SHA-pinned
  actions and a `govulncheck` job, but no `SECURITY.md` and no aggregate job. The rest moved
  to the "before v0.1.0" row; it only pays off once checks are required.
- **Aggregate check (deferred).** Required checks are the three matrix job names today. A skipped
  required check stays "Pending" and blocks merging ([docs][required-checks]). One `check`
  job with `needs:` on the matrix and `if: always()` lets the matrix change without
  editing the ruleset.
- **Up to date.** With one branch at a time it costs nothing. Dependabot rebases its own
  PRs; grouping keeps that to one PR a week.
- **Dependabot** ([options][dependabot-options]): `gomod` and `github-actions`, weekly,
  one group per ecosystem, `chore(deps)` commit titles, no labels (the repo has none). A 3-day cooldown applies by default. It updates SHA pins and
  their version comments. It does not bump the `toolchain` line in `go.mod`; do that by hand.
- **CodeQL rule.** The code-scanning rule is separate from status checks and doesn't apply
  to Dependabot PRs ([merge protection][codeql-merge]). Availability on the Free plan is
  unconfirmed; try it and drop it if the option is missing.
- **Skipped as not worth it yet:** OpenSSF Scorecard, workflow lockfiles
  ([GA 2026-10-28][actions-lock]; SHA pins plus Dependabot cover the same risk).

**0.3 prompt** (branch `chore/ci-hardening`, no OpenSpec). Item 1 already shipped in d4e7563, so
what remains is:

```text
CI hardening. No product code changes.

1. Pin every action in .github/workflows to a full commit SHA with a version
   comment. Look the SHAs up; don't guess them.
2. .github/workflows/govulncheck.yml: golang/govulncheck-action (pinned), on push
   to main, pull requests and a weekly cron, reading the Go version from go.mod.

Don't change repository settings; list the ones I must change by hand.

Done when: the PR's checks are green on all three systems, the govulncheck job
runs, and `make check` passes.
```

Status: done (d4e7563). Dependabot security updates and auto-delete of merged branches were
switched on 2026-11-16. Nothing else from this chore remains until the optional row above.

## 8. Release

**Shape:** tag `v*` → GoReleaser builds six cgo-free binaries → archives, `checksums.txt`,
attestation → GitHub Release. Homebrew and the install script read from that release.

- **GoReleaser** v2.20.2 (2026-11-12), [OSS][gr-releases]. Sketch; verify with
  `goreleaser check`:

  ```yaml
  version: 2
  builds:
    - env: [CGO_ENABLED=0]
      goos: [linux, darwin, windows]
      goarch: [amd64, arm64]
      flags: [-trimpath]
      ldflags: ["-s -w -X main.version={{ .Version }}"]
  archives:
    - formats: [tar.gz]
      format_overrides:
        - goos: windows
          formats: [zip]
  checksum:
    name_template: checksums.txt
  changelog:
    use: github
  ```

- **Version string.** No leading `v`: `mhz --version` prints `mhz 0.1.0`, matching
  `{{ .Version }}` and Homebrew's `version`. Tags stay `v0.1.0`.
- **Provenance.** [`actions/attest@v4`][gr-attest] with
  `subject-checksums: ./dist/checksums.txt`; free on public repos. Users verify with
  `gh attestation verify`.
- **If a cgo dependency ever appears (Linux only):** GoReleaser's split-and-merge is
  [Pro-only][gr-cgo]. Free route: one Ubuntu job, two build ids; darwin and windows stay
  `CGO_ENABLED=0`; linux/amd64 builds natively and linux/arm64 with `aarch64-linux-gnu-gcc`.
- **Homebrew.** Skip GoReleaser's Homebrew publishers. A cask installs a prebuilt, unsigned
  binary and needs a quarantine workaround. A formula that compiles from source has no
  downloaded binary to quarantine (inferred; the docs don't state it). Put this in
  `jacob-lineberry/homebrew-tap` as `Formula/mhz.rb`:

  ```ruby
  class Mhz < Formula
    desc "Curated internet radio for the terminal with an 80s car-stereo faceplate"
    homepage "https://github.com/jacob-lineberry/mhz"
    url "https://github.com/jacob-lineberry/mhz/archive/refs/tags/v0.1.0.tar.gz"
    sha256 "<sha256 of that tarball>"
    license "MIT"
    head "https://github.com/jacob-lineberry/mhz.git", branch: "main"

    depends_on "go" => :build

    def install
      system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}")
    end

    test do
      assert_match version.to_s, shell_output("#{bin}/mhz --version")
      assert_match "<one of your station names>", shell_output("#{bin}/mhz --stations")
    end
  end
  ```

  Since Homebrew 6.0, third-party taps must be [trusted][brew-trust]. The README command
  is the fully qualified `brew install jacob-lineberry/tap/mhz`, which trusts just this
  formula. Update `url` and `sha256` by hand per release at first.
- **install.sh** (repo root): POSIX `sh`, `set -eu`, body wrapped in `main` so a truncated
  download can't half-run. Map `uname` to archive names; resolve the latest tag with a
  `VERSION` override; fetch archive and `checksums.txt` with
  `curl --proto '=https' --tlsv1.2 -fsSL`; verify SHA-256 and fail closed; install to
  `${INSTALL_DIR:-$HOME/.local/bin}` without sudo; warn if that isn't on `PATH`.
- **Apple signing.** No planned channel requires a paid Developer ID. Files fetched by curl
  aren't quarantined and Go ad-hoc signs arm64 binaries. A browser-downloaded archive needs
  `xattr -d com.apple.quarantine`; say so in the README.
- **homebrew-core, later.** [Thresholds][brew-policy]: "at least 30 forks, 30 watchers or
  75 stars", and "at least 90 forks, 90 watchers or 225 stars for a self-submission by the
  repository owner". The repo must be at least 30 days old. Also needed: a stable tagged
  release, a source build, a real `test do` (hence `--stations`), passing on macOS and
  Linux. Realistic path: someone else submits it after 75 stars.

| Channel (R3, later) | How | Needs first |
|---|---|---|
| Scoop | GoReleaser [`scoops`][gr-scoop] to your own bucket repo | zip archives (already planned). Main bucket wants about 500 stars |
| winget | GoReleaser [`winget`][gr-winget] opens a PR to `winget-pkgs`; zip with a portable nested installer is accepted | A fork of `winget-pkgs` and a token; passes their validation scans |
| AUR | GoReleaser [`aurs`][gr-aur] for `mhz-bin`, `aur_sources` for a source package | AUR account and a passphrase-less SSH key as a secret |
| nixpkgs | Hand-written `buildGoModule` package in `pkgs/by-name` | A maintainer entry; self-submission is discouraged until there are users. GoReleaser's `nix` publisher only targets a NUR |

**R1 prompt** (branch `chore/release-pipeline`):

```text
Add the release pipeline. No product code changes.

1. .goreleaser.yaml from the sketch in docs/roadmap.md §8. Run `goreleaser check`
   and a local `goreleaser release --snapshot --clean`; show me the dist/ listing.
2. .github/workflows/release.yml: runs on tags v*; top-level permissions
   contents: read; the release job adds contents: write, id-token: write and
   attestations: write; goreleaser-action and actions/attest pinned to SHAs;
   attest dist/checksums.txt.
3. ci.yml: run `goreleaser check` on pull requests.
4. README: a Releases section saying how to verify a download.

List the settings I must change by hand (tag ruleset, immutable releases).
Don't create a tag or a release.

Done when: the snapshot build produces six archives plus checksums.txt, each
binary prints the snapshot version with --version, and `make check` passes.
```

**R2 prompt** (branch `chore/install-and-tap`, after v0.1.0 exists):

```text
Add install.sh and the Homebrew formula text. No product code changes.

1. install.sh at the repo root per docs/roadmap.md §8. Test it against the real
   v0.1.0 release into a temp INSTALL_DIR on this machine, and show a failing
   run with a corrupted checksum.
2. CI: run shellcheck on install.sh.
3. packaging/homebrew/mhz.rb: the formula from §8 with the real sha256. I copy it
   to my tap repository myself.
4. README: install section with the script one-liner, the fully qualified brew
   command, and manual download notes for Windows and for macOS quarantine.

Done when: the script installs a working binary here, refuses a bad checksum, and
shellcheck is clean.
```

**R3 prompt** (branch `chore/package-channels`):

```text
Add <Scoop | winget | AUR> publishing to .goreleaser.yaml, one channel per PR.
Read the current GoReleaser docs page for that publisher first. List every secret
and external repository I must create. Show `goreleaser check` and a snapshot run.
Don't publish anything.
```

## 9. Visual brief

Moved to [docs/design/faceplate.md](design/faceplate.md): mood, palette, typography, the
80x24 layout, size tiers and what to borrow. Changes 6, 7, 8 and 10 read it, and the
`openspec/config.yaml` context points every visual change at `docs/design/`.

## 10. Risks and open questions

**Risks**

- MP3 decoder: one option is archived and the others are young; S1 may end with "fork it and own it".
- oto's single fixed sample rate may exclude or complicate some stations.
- Nightride FM hasn't agreed to anything; renaming their stations could read as passing off without clear credit.
- Stream URLs change; an embedded list goes stale between releases, and user config is the only hot fix.
- CI can't hear; audio regressions surface only in your hands-on step.
- Float math in springs and FFT can differ between arm64 and amd64 and break goldens.
- macOS and Windows arm64 playback are untested until you have the hardware.
- WSLg audio may crackle or lag in ways real systems don't; don't tune buffers on WSL alone.
- Linux without PulseAudio or pipewire-pulse relies on oto's runtime ALSA fallback, untested here.
- ICY titles are untrusted input to a terminal; change 3 must sanitize them.
- Pure-Go AAC is LC-only and GPL/LGPL; many stations are HE-AAC only.
- Dependabot won't bump the Go toolchain line.
- The Fable weekly share may run out mid-project; the fallback is Opus at XHigh.
- homebrew-core needs 225 stars for a self-submission; don't plan around it.
- Change 11 grew: it now builds the pixel buffer, encoder and rain scene as well as the toggle. If its proposal passes 12 tasks, split the scene work into its own change first.
- The earlier rain implementation is gone. Change 11 starts from the values in its prompt, which you never approved in stills.
- `openspec/` exists only on this machine; back it up, or a lost disk loses every spec.

**Decided** (2026-11-16): 30 FPS frame tick; autoplay the last station; TOML config; s then 1–6 stores a
preset; versions print without `v`; the earlier rain branch was deleted (2026-11-16) and the rain scene is rebuilt in change 11.

**Open questions**

- Is `mhz` free as a package name on Homebrew, AUR, nixpkgs, winget and Scoop? (Unchecked.)
- Your station names, frequencies and ident lines?
- Text idents only, or audio stingers later?
- What surrounds the faceplate on very large terminals?
- A reduced-motion switch?

## Sources

Checked 2026-11-16. Stream facts come from direct `curl` probes; repository facts from
`gh api`, `go list -m` and the working tree.

[oto]: https://github.com/ebitengine/oto
[go-mp3]: https://github.com/hajimehoshi/go-mp3
[tphakala/go-mp3]: https://github.com/tphakala/go-mp3
[llehouerou/go-mp3]: https://github.com/llehouerou/go-mp3
[pion/opus]: https://github.com/pion/opus
[cliamp]: https://github.com/bjarneo/cliamp
[nightride]: https://nightride.fm/
[plaza]: https://plaza.one/
[somafm-tos]: https://somafm.com/contact/tos.html
[cc-model]: https://code.claude.com/docs/en/model-config.md
[cc-cache]: https://code.claude.com/docs/en/prompt-caching.md
[cc-fast]: https://code.claude.com/docs/en/fast-mode.md
[claude-pricing]: https://claude.com/pricing
[dependabot-options]: https://docs.github.com/en/code-security/dependabot/working-with-dependabot/dependabot-options-reference
[dependabot-automerge]: https://docs.github.com/en/code-security/dependabot/working-with-dependabot/automating-dependabot-with-github-actions
[codeql-build]: https://docs.github.com/en/code-security/reference/code-scanning/codeql/build-options-for-compiled-languages
[codeql-merge]: https://docs.github.com/en/code-security/concepts/code-scanning/merge-protection
[required-checks]: https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/collaborating-on-repositories-with-code-quality-features/troubleshooting-required-status-checks
[actions-lock]: https://github.blog/changelog/2026-10-28-github-actions-workflow-lockfiles-are-generally-available/
[gr-releases]: https://github.com/goreleaser/goreleaser/releases/latest
[gr-deprecations]: https://goreleaser.com/deprecations/
[gr-cgo]: https://goreleaser.com/limitations/cgo/
[gr-attest]: https://goreleaser.com/customization/attestations/
[gr-scoop]: https://goreleaser.com/customization/scoop/
[gr-winget]: https://goreleaser.com/customization/winget/
[gr-aur]: https://goreleaser.com/customization/aur/
[brew-policy]: https://docs.brew.sh/Package-Acceptance-Policy
[brew-trust]: https://docs.brew.sh/Tap-Trust
