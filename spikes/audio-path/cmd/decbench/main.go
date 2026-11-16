// Command decbench decodes an MP3 file with each candidate decoder and
// reports speed, output length and failures. Damage flags simulate what a
// live stream does to a decoder: joining mid-frame, flipped bytes, and
// splices where bytes went missing.
//
//	decbench -in cap.mp3 [-dec all] [-skip 777] [-flip 1e-4] [-splice 200000:3000] [-pcm out/]
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/cpu"
	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/dec"
)

func main() {
	in := flag.String("in", "", "MP3 file")
	which := flag.String("dec", "all", "decoder name or all")
	skip := flag.Int("skip", 0, "drop this many leading bytes (join mid-frame)")
	flip := flag.Float64("flip", 0, "probability each byte is replaced with a random byte")
	splice := flag.String("splice", "", "every:len — every N bytes, delete len bytes")
	seed := flag.Uint64("seed", 1, "PRNG seed for damage")
	pcmDir := flag.String("pcm", "", "write s16le PCM per decoder into this directory")
	flag.Parse()
	runtime.GOMAXPROCS(1)

	raw, err := os.ReadFile(*in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data := damage(raw, *skip, *flip, *splice, *seed)
	names := dec.Names
	if *which != "all" {
		names = []string{*which}
	}
	fmt.Printf("input=%s bytes=%d used=%d skip=%d flip=%g splice=%q\n", filepath.Base(*in), len(raw), len(data), *skip, *flip, *splice)
	fmt.Printf("%-12s %6s %10s %9s %9s %9s %8s  %s\n", "decoder", "rate", "frames", "audio_s", "wall_ms", "cpu_ms", "x_rt", "result")
	for _, n := range names {
		run(n, data, *pcmDir)
	}
}

func run(name string, data []byte, pcmDir string) {
	var out io.Writer = io.Discard
	if pcmDir != "" {
		_ = os.MkdirAll(pcmDir, 0o755)
		f, err := os.Create(filepath.Join(pcmDir, name+".s16"))
		if err != nil {
			panic(err)
		}
		defer f.Close()
		out = f
	}
	var (
		rate   int
		frames int64
		result = "eof"
	)
	runtime.GC()
	c0, w0 := cpu.Time(), time.Now()
	done := make(chan struct{})
	go func() {
		select {
		case <-done:
		case <-time.After(3 * time.Minute):
			fmt.Printf("%-12s HANG: no EOF after 3 min (frames so far unknown)\n", name)
			os.Exit(2)
		}
	}()
	defer close(done)
	func() {
		defer func() {
			if r := recover(); r != nil {
				result = fmt.Sprintf("PANIC: %v", r)
			}
		}()
		s, err := dec.Open(name, bytes.NewReader(data))
		if err != nil {
			result = "open error: " + err.Error()
			return
		}
		rate = s.SampleRate()
		buf := make([]byte, 16<<10)
		for {
			n, err := s.Read(buf)
			frames += int64(n / 4)
			_, _ = out.Write(buf[:n])
			if err == io.EOF {
				return
			}
			if err != nil {
				result = "error: " + err.Error()
				return
			}
		}
	}()
	wall, c := time.Since(w0), cpu.Time()-c0
	audio := 0.0
	if rate > 0 {
		audio = float64(frames) / float64(rate)
	}
	xrt := 0.0
	if c > 0 {
		xrt = audio / c.Seconds()
	}
	fmt.Printf("%-12s %6d %10d %9.2f %9.0f %9.0f %8.0f  %s\n", name, rate, frames, audio, ms(wall), ms(c), xrt, result)
}

func ms(d time.Duration) float64 { return float64(d) / 1e6 }

func damage(raw []byte, skip int, flip float64, splice string, seed uint64) []byte {
	d := append([]byte(nil), raw[min(skip, len(raw)):]...)
	rng := rand.New(rand.NewPCG(seed, 0x6d687a))
	if flip > 0 {
		for i := range d {
			if rng.Float64() < flip {
				d[i] = byte(rng.UintN(256))
			}
		}
	}
	if splice != "" {
		a, b, _ := strings.Cut(splice, ":")
		every, _ := strconv.Atoi(a)
		cut, _ := strconv.Atoi(b)
		var o []byte
		for i := 0; i < len(d); i += every + cut {
			o = append(o, d[i:min(i+every, len(d))]...)
		}
		d = o
	}
	return d
}
