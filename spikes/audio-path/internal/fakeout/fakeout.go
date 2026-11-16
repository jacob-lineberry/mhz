// Package fakeout is a test double for the audio device.
//
// The app should depend on the Output and Player interfaces below, never on
// *oto.Context directly. *oto.Player already satisfies Player; Oto adapts
// *oto.Context to Output in three lines. Tests use Device instead: it pulls
// PCM exactly the way oto's mux does (non-blocking reads, short reads become
// silence), but only when the test calls Advance, so time is fake and no
// sound card is opened.
package fakeout

import (
	"io"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
)

// Output is the slice of oto.Context the app uses.
type Output interface {
	NewPlayer(r io.Reader) Player
	Err() error
}

// Player is the slice of *oto.Player the app uses.
type Player interface {
	Play()
	Pause()
	IsPlaying() bool
	SetVolume(v float64)
	BufferedSize() int
	Err() error
}

// Oto adapts a real context.
type Oto struct{ C *oto.Context }

// NewPlayer implements Output.
func (o Oto) NewPlayer(r io.Reader) Player { return o.C.NewPlayer(r) }

// Err implements Output.
func (o Oto) Err() error { return o.C.Err() }

// Device is a fake Output at a fixed rate, s16le stereo.
type Device struct {
	Rate    int
	mu      sync.Mutex
	players []*FakePlayer
	err     error
}

// NewPlayer implements Output.
func (d *Device) NewPlayer(r io.Reader) Player {
	d.mu.Lock()
	defer d.mu.Unlock()
	p := &FakePlayer{r: r, vol: 1}
	d.players = append(d.players, p)
	return p
}

// Err implements Output. Set FailWith to simulate a missing device.
func (d *Device) Err() error { d.mu.Lock(); defer d.mu.Unlock(); return d.err }

// FailWith makes Err return err.
func (d *Device) FailWith(err error) { d.mu.Lock(); d.err = err; d.mu.Unlock() }

// Advance plays dt of audio: every playing player is asked for exactly
// that many frames. Frames the source could not supply are recorded as
// silence and counted as underrun.
func (d *Device) Advance(dt time.Duration) {
	frames := int(dt.Seconds() * float64(d.Rate))
	d.mu.Lock()
	ps := append([]*FakePlayer(nil), d.players...)
	d.mu.Unlock()
	for _, p := range ps {
		p.pull(frames)
	}
}

// FakePlayer records what the device "heard".
type FakePlayer struct {
	mu       sync.Mutex
	r        io.Reader
	playing  bool
	vol      float64
	Heard    []byte // s16le stereo, after volume
	Underrun int    // frames of silence inserted because the source was empty
	err      error
}

func (p *FakePlayer) pull(frames int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.playing || p.err != nil {
		return
	}
	want := frames * 4
	buf := make([]byte, want)
	got := 0
	for got < want {
		n, err := p.r.Read(buf[got:])
		got += n
		if err != nil && err != io.EOF {
			p.err = err
		}
		if n == 0 || err != nil {
			break
		}
	}
	got -= got % 4
	for i := 0; i < got; i += 2 {
		v := float64(int16(uint16(buf[i])|uint16(buf[i+1])<<8)) * p.vol
		u := uint16(int16(v))
		buf[i], buf[i+1] = byte(u), byte(u>>8)
	}
	clear(buf[got:])
	p.Underrun += (want - got) / 4
	p.Heard = append(p.Heard, buf...)
}

func (p *FakePlayer) Play()               { p.mu.Lock(); p.playing = true; p.mu.Unlock() }
func (p *FakePlayer) Pause()              { p.mu.Lock(); p.playing = false; p.mu.Unlock() }
func (p *FakePlayer) IsPlaying() bool     { p.mu.Lock(); defer p.mu.Unlock(); return p.playing }
func (p *FakePlayer) SetVolume(v float64) { p.mu.Lock(); p.vol = v; p.mu.Unlock() }
func (p *FakePlayer) BufferedSize() int   { return 0 }
func (p *FakePlayer) Err() error          { p.mu.Lock(); defer p.mu.Unlock(); return p.err }
