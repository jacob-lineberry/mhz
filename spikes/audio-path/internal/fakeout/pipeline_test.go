package fakeout_test

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/dec"
	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/fakeout"
	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/icy"
	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/ring"
)

const metaint = 16000

// icyServer serves testdata/tone440.mp3 with ICY metadata interleaved, the
// way Icecast does, then closes the connection.
func icyServer(t *testing.T) *httptest.Server {
	t.Helper()
	mp3, err := os.ReadFile("../../testdata/tone440.mp3")
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Icy-MetaData") != "1" {
			http.Error(w, "want Icy-MetaData", 400)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Header().Set("icy-metaint", fmt.Sprint(metaint))
		var b bytes.Buffer
		meta := "StreamTitle='Test - Tone';"
		for a := mp3; len(a) > 0; {
			n := min(metaint, len(a))
			b.Write(a[:n])
			a = a[n:]
			if n == metaint {
				l := (len(meta) + 15) / 16
				b.WriteByte(byte(l))
				b.WriteString(meta)
				b.Write(make([]byte, l*16-len(meta)))
			}
		}
		_, _ = w.Write(b.Bytes())
	}))
}

func TestPipelineThroughFakeDevice(t *testing.T) {
	for _, name := range dec.Names {
		t.Run(name, func(t *testing.T) {
			srv := icyServer(t)
			defer srv.Close()
			req, _ := http.NewRequest("GET", srv.URL, nil)
			req.Header.Set("Icy-MetaData", "1")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			var title atomic.Value
			title.Store("")
			ir := icy.NewReader(resp.Body, metaint, func(raw string) { title.Store(icy.Title(raw)) })
			s, err := dec.Open(name, ir)
			if err != nil {
				t.Fatal(err)
			}
			rate := s.SampleRate()
			rb := ring.New(10*rate*4, 4, time.Now)
			done := make(chan error, 1)
			go func() {
				_, err := io.Copy(writerFunc(func(p []byte) (int, error) { rb.Write(p); return len(p), nil }), s)
				done <- err
			}()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("decoder did not finish")
			}

			dev := &fakeout.Device{Rate: rate}
			var out fakeout.Output = dev
			p := out.NewPlayer(rb)
			p.Play()
			for range 40 {
				dev.Advance(100 * time.Millisecond)
			}
			fp := p.(*fakeout.FakePlayer)
			if got, want := len(fp.Heard), 4*rate*4; got != want {
				t.Fatalf("heard %d bytes, want %d", got, want)
			}
			if fp.Underrun != 0 {
				t.Fatalf("underrun %d frames", fp.Underrun)
			}
			hz, rms := analyze(fp.Heard[rate*4:], rate) // skip the first second
			if math.Abs(hz-440) > 2 {
				t.Errorf("frequency %.1f Hz, want 440", hz)
			}
			if math.Abs(rms-0.5/math.Sqrt2) > 0.02 {
				t.Errorf("rms %.3f, want %.3f", rms, 0.5/math.Sqrt2)
			}
			if got := title.Load(); got != "Test - Tone" {
				t.Errorf("title %q", got)
			}

			p.Pause()
			before := len(fp.Heard)
			dev.Advance(time.Second)
			if len(fp.Heard) != before {
				t.Error("paused player consumed audio")
			}
		})
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// analyze returns the left channel's zero-crossing frequency and RMS.
func analyze(pcm []byte, rate int) (hz, rms float64) {
	n := len(pcm) / 4
	cross, sum := 0, 0.0
	prev := 0.0
	for i := range n {
		v := float64(int16(uint16(pcm[4*i])|uint16(pcm[4*i+1])<<8)) / 32768
		sum += v * v
		if i > 0 && (prev < 0) != (v < 0) {
			cross++
		}
		prev = v
	}
	return float64(cross) / 2 / (float64(n) / float64(rate)), math.Sqrt(sum / float64(n))
}
