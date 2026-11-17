# 0001. Audio path: decoder, output, buffering

Status: Proposed (draft from spike S1, 2026-11-16). Decoder and sample rate decided
2026-11-16 (see "Decision"). WSL2 and native Windows 11 measured; macOS not yet, and no
listening test yet. Stays Proposed until both are done.

Evidence tags: **[measured]** = run here, command in "Reproduce"; **[source]** = read in the
dependency's source at the pinned version; **[assumed]** = not checked.

## Context

mhz plays curated internet-radio MP3 streams in a Bubble Tea TUI. The roadmap rules out
cgo, so releases, CI and CodeQL stay simple. Change 1 (`add-stream-playback`) needs a
decoder, an output library and a buffering shape. Spike code is in `spikes/audio-path/` on
branch `spike/audio-path`.

Machine: Ryzen 9 9950X3D, WSL2 (kernel 6.18, WSLg PulseAudio over RDP) and the same box's
native Windows 11 (WASAPI), reached by running a `GOOS=windows` exe through WSL interop.
Go 1.26.8 for mhz; the spike module is Go 1.27 (see Q1).

## Answers

### Q1. Which MP3 decoder

| | [hajimehoshi/go-mp3] v0.3.4 | [llehouerou/go-mp3] v1.5.0 | [tphakala/go-mp3] v0.4.0 |
|---|---|---|---|
| Lineage | PDMP3 port | Fork of hajimehoshi | Function-by-function port of minimp3 (CC0), tested bit-exact against a pinned minimp3 build on amd64 and arm64 [source] |
| Status | Archived 2023, 776★ | Active, 0★, 1 maintainer | Active (v0.1.0 2026-09, v0.4.0 2026-11-15), 1★, 1 maintainer. 12 open issues, all about the encoder [measured, `gh`] |
| License | Apache-2.0 | Apache-2.0 | MIT (and the `tphakala/simd` dependency is MIT) |
| API | `io.Reader` of s16le stereo | Same, plus seeking | `DecodeFrame([]byte, []float32)`. mhz owns a ~100-line streaming adapter (`internal/dec/dec.go`) |
| Go floor | 1.14 | 1.25 | **1.27** (mhz is on 1.26) [measured: `go get` switched the toolchain] |
| Extra deps | none | oto v3.4.0 (example). **`go mod tidy` resolved oto to v3.4.0, which needs cgo + ALSA headers on Linux** unless mhz pins v3.5.1 [measured] | `tphakala/simd` (assembly), `x/sys`. These come from the encoder in the same package; the decoder (`internal/dec` + `internal/bits`, 2.4k lines) imports neither [source] |
| CPU for 30.7 min of Nightride at 320 kbps (73.7 MB), `GOMAXPROCS=1` | 13.97 s = **132× real time** | 3.98 s = **463×** | 3.14 s = **586×** [measured]. The live pipeline's CPU is in Q2 |
| Output vs minimp3, 70,543 blocks | Identical to tphakala within ±1 LSB, except the first frame [measured, `pcmcmp`, `blockdiff`]. The 46.8 dB overall SNR is that one frame plus 1-LSB rounding in near-silence | Same as hajimehoshi (not byte-identical to it) | Reference |
| Stream join | **Two garbage frames (~52 ms) on every connect: peaks at full scale (32767), sample steps 3–4× normal** [measured at offsets 0, 1, 500, 1000]. Expected to be heard as a click [assumed; listening test pending] | Same | First frame quieter than the music; no overshoot. One frame fewer output (minimp3 drops frames with no bit reservoir) |
| Join mid-frame (20 offsets) | 20/20 decode to the end | 20/20 | 20/20 [measured] |
| Random bytes 1e-4 (3 seeds) | **Fatal error in 2/3** (`is_pos was too big`, `MPEG version 2.5 is not supported`) | **Fatal error in 1/3** | 3/3 to the end, 99.5–99.9% of audio [measured] |
| Random bytes 1e-3 (3 seeds) | **Fatal in 3/3**, one at open (`free bitrate format is not supported`) | **Fatal in 3/3** | 3/3 to the end, 98.2–99.1% [measured] |
| Splice (bytes lost every 400 kB: 1, 100, 777, 3000) | **Fatal on the 1-byte splice** (`only layer3 … is supported`) | 4/4 | 4/4 [measured] |
| ICY metadata left in the audio (a demux bug or a server quirk) | **Fatal on 3 of 6 stations' samples** | **Fatal on 3 of 6** | 6/6 to the end [measured] |
| MPEG 2.5, free format | Rejects both (fatal) [source] | Supports 2.5, rejects free format [source] | Supports both [source] |

A fatal error from the PDMP3 decoders ends the `Read` loop. mhz would have to detect it, build
a new decoder on the same reader and accept a gap plus another start click. tphakala resyncs
inside `DecodeFrame` and returned no error in any of these runs.

### Q2. Does oto v3.5.1 play cleanly without cgo, and with what buffer

- **Builds:** `CGO_ENABLED=0` for linux/{amd64,arm64}, darwin/{amd64,arm64} and
  windows/{amd64,arm64} [measured]. With cgo on and no ALSA headers, `go test -race` still
  builds (oto 3.5 loads libasound with purego) [measured].
- **Plays:** 30 min each on WSL2 (PulseAudio) and native Windows (WASAPI) with zero ring
  starvation, zero gaps and heap at 13–25 MB [measured, `-vol 0`]. Quit (pause, cancel,
  decoder goroutine exits) took 0.13 ms on WSL and 1.2 ms on Windows. **Not listened to yet.**
- **Driver defaults** [source]: PulseAudio 100 ms requested (70 ms granted on WSLg
  [measured]), WASAPI 50 ms, ALSA 2×1024 frames (46 ms), macOS 4×1536 frames (139 ms). oto
  also keeps a player buffer of 0.5 s by default, refilled in 0.5 s chunks.
- **Underruns can't be tuned on WSLg.** A direct PulseAudio probe reported no underflow
  even at 5 ms requested / 3.7 ms granted, with 31 ms gaps between callbacks. The RDP sink
  absorbs everything [measured, `pulseprobe`]. Tuning needs native Windows and macOS, by ear.
- **What latency means here.** Pause and oto `SetVolume` act at the mixer, so they're heard
  after the device buffer alone (≤ 50–140 ms) [source]. Anything done *before* the ring is
  heard after ring + player buffer: 12–13 s on Nightride, because of its burst on connect.
  **So software gain (change 5) must be oto `Player.SetVolume` or applied on the read side
  of the ring, never at decode time.** The S3 tap must account for ring + player + device
  delay.
- **CPU:** the whole pipeline (TLS, ICY, decode, ring, oto, PulseAudio) uses **1.0% of one
  core** on WSL2 (60 s on Plaza, profiled). Most of it is goroutine wakeups (`futex`);
  decoding is about 0.15%. The 30-minute runs show 3.2% (WSL) and 1.8% (Windows) because they
  included a 5 ms polling probe that costs about 2.5% by itself [measured, `-cpuprofile`].
- **Missing device:** with no PulseAudio server and no libasound, `oto.NewContext`'s
  `ready` closes after 0.3 ms and `Err()` names both backends. Nothing hangs [measured,
  `devcheck`].

### Q3. Clock drift over 30 minutes

Both platforms played Nightride chillsynth for 1800 s at the same time, silent, with no
starvation, no gaps and no dropped audio [measured]. Drift is the slope of total buffered
audio (ring + oto player buffer) against wall time, logged once a second
[measured, `scripts/drift.py`]:

| Platform | Buffered, min–max over 30 min | Slope, full run | First half | Second half |
|---|---|---|---|---|
| Windows (WASAPI) | 12.57–12.97 s | −52 ms/h (−14 ppm) | +5 ppm | −15 ppm |
| WSL2 (WSLg) | 12.56–12.98 s | −96 ms/h (−27 ppm) | −79 ppm | −12 ppm |

The ±0.3 s wobble is network arrival burstiness, so the slopes are good to about ±50 ppm
over 30 minutes. **The buffer neither grows nor drains measurably.** At −27 ppm, Nightride's
~12.4 s burst would last over 100 hours, and a 2 s prebuffer about 20 hours.

Correction: an earlier 7-minute estimate said WSLg drained at about 600 ppm. That came from
counting oto's ring reads, which oto makes in 0.5 s chunks. `drift.py`'s arrival/consume
lines carry the same quantization and shouldn't be trusted over short spans; use buffered
level instead.

Policy proposed:

1. Read the network as fast as it arrives into a bounded ring sized to hold the burst
   (≥ 15 s at the stream's rate). Read never blocks oto; when empty it returns `0, nil`.
2. No drift compensation in change 1.
3. If the ring empties, re-prebuffer: that's S2's buffering state, the same path as a stall.
4. If the ring fills, stop reading the socket (TCP backpressure) instead of dropping audio.
   The server's per-client queue decides what happens next; S2 measures it.
5. Re-measure on macOS. Only if a real device shows > 100 ppm, add slow rate correction in
   a later change (drop or repeat one frame per N, or a ±0.1% resampler).

### Q4. Sample rates

All six candidate streams are **44.1 kHz stereo** (first decoded frame; laut.fm and
Frekvenci also declare `icy-sr: 44100`) [measured]. oto allows one context per process at
one rate and has no `Close` [source]. Every backend converts to the device rate itself:
WASAPI with `AUTOCONVERTPCM | SRC_DEFAULT_QUALITY`, PulseAudio and CoreAudio in the server
[source].

**Answer: open oto at 44.1 kHz. A non-44.1 kHz station is "unsupported" in change 1.**
Resampling waits until a station needs it. A user station at 48 kHz would hit that state,
not a crash.

### Q5. ICY

- In-band parsing works on all six streams, Nightride and Plaza included: titles arrive
  every `icy-metaint` = 16000 bytes on every server [measured]. Titles seen include UTF-8
  (`天気予報 - Light Above The Trees`, `—`), `StreamUrl` keys with query strings, and
  laut.fm sponsor text in `StreamTitle`.
- **No candidate answers `ICY 200 OK`.** Nightride and Frekvenci speak HTTP/2, the rest
  HTTP/1.1, and Go's client handles all of them [measured]. Plaza and laut.fm also serve
  HTTPS (`https://radio.plaza.one/mp3`; laut.fm redirects to an HTTPS host) [measured].
- Nightride's `icy-name` is an opaque id, as the roadmap noted.
- Demux code: `internal/icy` (~70 lines) plus a 1-byte-read test. cliamp's shim has the same
  shape.

### Q6. Test double for the output device

`internal/fakeout` [measured: `go test -race -count=3` passes for all three decoders]:

- The app depends on two small interfaces, `Output{NewPlayer(io.Reader) Player; Err()}` and
  `Player{Play; Pause; IsPlaying; SetVolume; BufferedSize; Err}`. `*oto.Player` already
  satisfies `Player`, and a three-line adapter wraps `*oto.Context`.
- `fakeout.Device.Advance(d)` pulls exactly `d × rate` frames from each playing player,
  like oto's mixer does. Short reads become silence and are counted as `Underrun`, and
  what was "heard" is recorded. Tests drive time; nothing opens a sound card.
- The end-to-end test runs an `httptest` server with ICY metadata, then ICY demux, decoder,
  ring and fake device. It asserts 440 Hz ±2, RMS ±0.02, the title, no underrun, and that
  pause stops consumption. The fixture `testdata/tone440.mp3` (80 kB, 5 s) comes from
  tphakala's pure-Go encoder, so fixtures need no external tools.

## Options considered

1. **tphakala/go-mp3 as a dependency.** Best robustness, fastest, no start click, MIT.
   Costs: Go 1.27 for mhz, the encoder and `simd` assembly linked in, a young project with
   one maintainer.
2. **Vendor tphakala's decoder** (`internal/dec` and `internal/bits`, 2,437 lines, no imports
   outside themselves [source], MIT)
   into `internal/mp3` with its license. Same behavior, stays on Go 1.26, no `simd`. mhz
   owns updates; there are no decoder bugs open upstream today.
3. **llehouerou/go-mp3.** A drop-in `io.Reader`, active. But it has fatal errors on
   corruption, the start click, and a dependency on oto v3.4.0 that pulls cgo back in
   unless pinned.
4. **hajimehoshi/go-mp3.** Archived, the slowest, and the least robust: it dies on a
   1-byte splice.
5. **ffmpeg as a subprocess** (cliamp's route for AAC and Opus). Rejected for MP3: an
   external runtime dependency, against the no-cgo, single-binary goal.

## Recommendation

- **Decoder: tphakala/go-mp3 v0.4.0 (option 1), with option 2 as the fallback** if you'd
  rather not move to Go 1.27 or if upstream stalls. Bump mhz to `go 1.27` / `toolchain
  go1.27.x` in change 1.
- **Output: oto v3.5.1, pinned exactly** (v3.4.0 needs cgo on Linux and `go mod tidy` can
  pick it). One context at 44.1 kHz, s16le stereo, driver-default device buffer until
  listening tests say otherwise. Player buffer: start at oto's default 500 ms;
  [assumed] smaller is fine because the ring holds the slack.
- **Shape:** one goroutine owns HTTP → ICY → decoder → ring and stops on context
  cancellation. The ring's `Read` never blocks. Gain goes through `Player.SetVolume`. The
  app sees oto only through the `Output`/`Player` interfaces.

## Decision

Made by the maintainer, 2026-11-16:

- **Decoder: tphakala/go-mp3 v0.4.0 as a dependency (option 1).** mhz moves to Go 1.27 in
  change 1. Vendoring (option 2) stays the fallback if upstream stalls.
- **44.1 kHz only for change 1.** Streams at other rates show as unsupported; no resampling.
- The output, shape and pinning follow the recommendation above.

## Consequences

- mhz needs Go 1.27, and Dependabot won't bump the toolchain line. CI reads
  `go-version-file`, so no workflow edits are needed. The pinned golangci-lint v2.15.0 is
  built with go1.27.0 and lints the Go 1.27 spike module [measured].
- Change 1's approved dependencies: `github.com/ebitengine/oto/v3` v3.5.1 and
  `github.com/tphakala/go-mp3` v0.4.0. That brings in `ebitengine/purego`,
  `jfreymuth/pulse`, `tphakala/simd` and `golang.org/x/sys` indirectly. No cgo.
- CodeQL and the release stay as they are (no cgo).
- Change 5 must apply gain after the ring. S3 must measure ring + player + device delay.
- S2 inherits: re-prebuffer on empty, backpressure on full, and
  servers' burst on connect, which varies widely. Estimated from 6 s captures [measured]:
  Nightride ~13 s, laut.fm ~17 s, Fantasy ~15 s, Plaza ~3.6 s, Frekvenci ~1.5 s. Time from
  launch to Play was 0.6–0.8 s with a 2 s prebuffer on Nightride.
- tphakala's adapter keeps 16 KiB of input ahead of the decoder so minimp3 can confirm sync
  over 10 frame headers [source]. That costs 0.4 s of input at 320 kbps and 1 s at 128 kbps
  before the first frame, unless the server bursts.

## Open questions for you

1. ~~Go 1.27 with tphakala as a dependency, or vendor its decoder and stay on 1.26?~~
   Decided: dependency, Go 1.27.
2. Listening tests (see "Listening tests" below) on WSL2 and Windows: any clicks, crackle or dropouts?
   Do you hear the start click with the PDMP3 decoders?
3. macOS when the MacBook arrives: the same 30-minute run, with drift and a listening test.
4. ~~Is "44.1 kHz only, other rates unsupported" acceptable for change 1?~~ Decided: yes.

## Listening tests (for you)

Build once from `spikes/audio-path/` on branch `spike/audio-path`:

```sh
CGO_ENABLED=0 go build -o /tmp/play ./cmd/play
GOOS=windows CGO_ENABLED=0 go build -o /tmp/play.exe ./cmd/play   # runs on Windows through WSL interop
```

| # | Command | Listen for |
|---|---|---|
| 1 | `/tmp/play.exe -vol 0.7 -watch=false -dur 10m` | Native Windows, default 50 ms buffer: any click, crackle or dropout |
| 2 | `/tmp/play.exe -vol 0.7 -watch=false -dur 3m -devbuf 20ms`, then `-devbuf 10ms` | The smallest buffer that is still clean |
| 3 | `/tmp/play -vol 0.7 -watch=false -dur 10m` | WSL2 through WSLg: the same, plus any lag or warble |
| 4 | `/tmp/play.exe -vol 0.7 -dur 8s -dec hajimehoshi`, then `-dec tphakala` | A click or burst in the first tenth of a second with hajimehoshi but not with tphakala |
| 5 | `/tmp/play.exe -vol 0.7 -dur 30m -url https://radio.plaza.one/mp3 -csv plaza-win.csv` | A long listen on a second broadcaster; then `python3 -I scripts/drift.py plaza-win.csv` |

Each run prints a summary line; `starves=0 gaps=0` means the pipeline itself never ran dry.

## Reproduce

All from `spikes/audio-path/` on branch `spike/audio-path`; full commands in its
`README.md`.

| Measurement | Command |
|---|---|
| Station headers, rates, titles | `curl -sS -m 6 -H 'Icy-MetaData: 1' -D - -o p.bin <url>`, then `go run ./cmd/icystrip <p.bin >p.mp3 && go run ./cmd/decbench -in p.mp3` |
| Decoder CPU, agreement, damage | `sh scripts/decoders.sh cap.mp3 /tmp/s1` on a 30-min capture made with `curl -m 1830 -o cap.mp3 https://stream.nightride.fm/nightride.mp3` |
| Start click | `go run ./cmd/decbench -in head.mp3 -skip 500 -pcm out/`, then compare the first frames' peaks |
| Drift, starvation, CPU | `CGO_ENABLED=0 go build -o play ./cmd/play && ./play -url https://stream.nightride.fm/chillsynth.mp3 -dur 30m -csv run.csv; python3 -I scripts/drift.py run.csv` (Windows: build with `GOOS=windows` and run `play.exe`) |
| WSLg underflow | `go run ./cmd/pulseprobe -each 20s` |
| Missing device | `PULSE_SERVER=unix:/nonexistent go run ./cmd/devcheck` |
| Fake device | `CGO_ENABLED=0 go test ./... && go test -race -count=3 ./internal/...` |

[hajimehoshi/go-mp3]: https://github.com/hajimehoshi/go-mp3
[llehouerou/go-mp3]: https://github.com/llehouerou/go-mp3
[tphakala/go-mp3]: https://github.com/tphakala/go-mp3
