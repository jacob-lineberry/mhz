// Command gentone writes a stereo sine-tone MP3 using tphakala/go-mp3's
// pure-Go encoder, so test fixtures need no external tools.
//
//	gentone -hz 440 -sec 5 -rate 44100 -kbps 128 -o testdata/tone440.mp3
package main

import (
	"flag"
	"math"
	"os"

	mp3 "github.com/tphakala/go-mp3"
)

func main() {
	hz := flag.Float64("hz", 440, "tone frequency")
	sec := flag.Float64("sec", 5, "duration")
	rate := flag.Int("rate", 44100, "sample rate")
	kbps := flag.Int("kbps", 128, "bitrate")
	out := flag.String("o", "tone.mp3", "output file")
	flag.Parse()

	e, err := mp3.NewEncoder(mp3.EncoderConfig{SampleRate: *rate, Channels: 2, Bitrate: *kbps * 1000})
	if err != nil {
		panic(err)
	}
	total := int(*sec * float64(*rate))
	var stream []byte
	l, r := make([]float32, mp3.FrameSize), make([]float32, mp3.FrameSize)
	for pos := 0; pos < total; pos += mp3.FrameSize {
		n := min(mp3.FrameSize, total-pos)
		for i := range n {
			v := float32(0.5 * math.Sin(2*math.Pi*(*hz)*float64(pos+i)/float64(*rate)))
			l[i], r[i] = v, v
		}
		stream, err = e.EncodeFrame(stream, [][]float32{l[:n], r[:n]})
		if err != nil {
			panic(err)
		}
	}
	if stream, err = e.EncodeFrame(stream, nil); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, stream, 0o644); err != nil {
		panic(err)
	}
}
