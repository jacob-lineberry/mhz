// Package dec puts the three candidate MP3 decoders behind one streaming
// interface: interleaved signed 16-bit little-endian stereo PCM.
package dec

import (
	"errors"
	"fmt"
	"io"
	"math"

	hajime "github.com/hajimehoshi/go-mp3"
	lleh "github.com/llehouerou/go-mp3"
	tpk "github.com/tphakala/go-mp3"
)

// Names lists the decoders in a stable order.
var Names = []string{"hajimehoshi", "llehouerou", "tphakala"}

// Stream is decoded PCM: s16le, 2 channels, at SampleRate.
type Stream interface {
	io.Reader
	SampleRate() int
}

// Open starts decoding r with the named decoder. All three read the first
// frame before returning, so Open blocks until audio arrives.
func Open(name string, r io.Reader) (Stream, error) {
	switch name {
	case "hajimehoshi":
		return hajime.NewDecoder(r)
	case "llehouerou":
		return lleh.NewDecoderWithOptions(r, lleh.DecoderOptions{Gapless: false})
	case "tphakala":
		return newTP(r)
	}
	return nil, fmt.Errorf("unknown decoder %q", name)
}

// minBuffered is how much input the tphakala adapter keeps ahead of the
// decoder. minimp3 wants several frames to confirm sync; 16 KiB is its
// documented streaming recommendation and holds 11 frames at 320 kbps.
const minBuffered = 16 << 10

// tpStream adapts tphakala's frame-at-a-time API to an io.Reader. This is
// the extra code mhz would own if it picked tphakala.
type tpStream struct {
	r    io.Reader
	d    *tpk.Decoder
	in   []byte
	off  int
	eof  bool
	pcm  []float32
	out  []byte
	rate int
	err  error
}

func newTP(r io.Reader) (*tpStream, error) {
	s := &tpStream{r: r, d: tpk.NewDecoder(), in: make([]byte, 0, 4*minBuffered), pcm: make([]float32, 1152*2)}
	if err := s.decodeOne(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *tpStream) SampleRate() int { return s.rate }

func (s *tpStream) fill() error {
	if s.off > 0 {
		n := copy(s.in, s.in[s.off:])
		s.in = s.in[:n]
		s.off = 0
	}
	for !s.eof && len(s.in) < minBuffered {
		if len(s.in) == cap(s.in) {
			s.in = append(s.in, 0)[:len(s.in)]
		}
		n, err := s.r.Read(s.in[len(s.in):cap(s.in)])
		s.in = s.in[:len(s.in)+n]
		if errors.Is(err, io.EOF) {
			s.eof = true
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (s *tpStream) decodeOne() error {
	for {
		if !s.eof && len(s.in)-s.off < minBuffered {
			if err := s.fill(); err != nil {
				return err
			}
		}
		data := s.in[s.off:]
		if len(data) == 0 {
			return io.EOF
		}
		n, info, err := s.d.DecodeFrame(data, s.pcm)
		if info.FrameBytes == 0 {
			return io.EOF
		}
		s.off += info.FrameBytes
		if err != nil && !errors.Is(err, tpk.ErrUnsupported) {
			return err
		}
		if n == 0 {
			continue
		}
		if s.rate == 0 {
			s.rate = info.SampleRate
		}
		s.out = appendS16(s.out[:0], s.pcm[:n*info.Channels], info.Channels)
		return nil
	}
}

func appendS16(dst []byte, pcm []float32, ch int) []byte {
	put := func(v float32) {
		x := math.Round(float64(v) * 32767)
		x = max(-32768, min(32767, x))
		u := uint16(int16(x))
		dst = append(dst, byte(u), byte(u>>8))
	}
	if ch == 1 {
		for _, v := range pcm {
			put(v)
			put(v)
		}
		return dst
	}
	for _, v := range pcm {
		put(v)
	}
	return dst
}

func (s *tpStream) Read(p []byte) (int, error) {
	for len(s.out) == 0 {
		if s.err != nil {
			return 0, s.err
		}
		if err := s.decodeOne(); err != nil {
			s.err = err
			return 0, err
		}
	}
	n := copy(p, s.out)
	s.out = s.out[n:]
	return n, nil
}
