# Spike S1: audio path

Throwaway research for [ADR-0001](../../docs/decisions/0001-audio-path.md). Own `go.mod`
(Go 1.27, because `tphakala/go-mp3` requires it); nothing in the main module imports this.

| Path | What |
|---|---|
| `internal/dec` | The three candidate decoders behind one `io.Reader` (s16le stereo). The `tphakala` adapter is the code mhz would own if it picks that decoder |
| `internal/icy` | In-band ICY metadata demuxer and `StreamTitle` parser |
| `internal/ring` | Non-blocking PCM FIFO between the decoder goroutine and oto, with starvation counters |
| `internal/fakeout` | Test double for the audio device (Q6) and an end-to-end test through it |
| `cmd/decbench` | Decode a file with each decoder; damage flags for mid-frame join, flipped bytes, splices |
| `cmd/pcmcmp` | Align two PCM files and report SNR and bad blocks |
| `cmd/play` | Live stream → ICY → decoder → ring → oto, with per-second CSV telemetry |
| `cmd/pulseprobe` | PulseAudio underflow check per latency (Linux) |
| `cmd/devcheck` | Open an oto context and report the error and time taken |
| `cmd/gentone` | Write a sine-tone MP3 with the pure-Go encoder (`testdata/`) |
| `cmd/icystrip` | Remove ICY metadata from a captured body |
| `scripts/decoders.sh` | The Q1 decoder comparison |

## Reproduce

```sh
cd spikes/audio-path
CGO_ENABLED=0 go test ./...                      # Q6: pipeline through the fake device
go test -race -count=3 ./internal/...

# Q1: capture 30 min with no ICY metadata, then compare decoders
curl -sS -m 1830 -A 'mhz-spike/0' -o /tmp/cap.mp3 https://stream.nightride.fm/nightride.mp3
sh scripts/decoders.sh /tmp/cap.mp3 /tmp/s1

# Q2/Q3: 30 min silent playback with telemetry (add -vol 1 to listen)
CGO_ENABLED=0 go build -o /tmp/play ./cmd/play
/tmp/play -url https://stream.nightride.fm/chillsynth.mp3 -dur 30m -csv /tmp/drift.csv

# Native Windows from WSL (runs through interop; WASAPI)
GOOS=windows CGO_ENABLED=0 go build -o /tmp/play.exe ./cmd/play && /tmp/play.exe -dur 30m -csv drift-win.csv

# Q2 on Linux: device-level underflow per latency
CGO_ENABLED=0 go build -o /tmp/pulseprobe ./cmd/pulseprobe && /tmp/pulseprobe -each 20s

# Q4/Q5: headers and sample rate of a station
curl -sS -m 6 -H 'Icy-MetaData: 1' -D - -o /tmp/p.bin https://stream.nightride.fm/nightride.mp3
go run ./cmd/icystrip -metaint 16000 </tmp/p.bin >/tmp/p.mp3 && go run ./cmd/decbench -in /tmp/p.mp3

# Missing device
PULSE_SERVER=unix:/nonexistent go run ./cmd/devcheck
```
