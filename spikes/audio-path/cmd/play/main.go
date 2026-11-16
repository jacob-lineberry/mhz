// Command play streams one station through oto and logs buffer telemetry
// once per second as CSV, for underrun, latency and clock-drift checks.
//
//	play -url https://stream.nightride.fm/nightride.mp3 -dec tphakala -dur 30m -csv run.csv [-vol 1]
//
// Pipeline: HTTP body -> ICY demux -> decoder goroutine -> ring -> oto player.
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ebitengine/oto/v3"

	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/cpu"
	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/dec"
	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/icy"
	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/ring"
)

const userAgent = "mhz-spike/0 (+https://github.com/jacob-lineberry/mhz)"

type counter struct {
	r io.Reader
	n atomic.Int64
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n.Add(int64(n))
	return n, err
}

func main() {
	url := flag.String("url", "https://stream.nightride.fm/nightride.mp3", "stream URL")
	decName := flag.String("dec", "tphakala", "decoder")
	devBuf := flag.Duration("devbuf", 0, "oto device BufferSize (0 = driver default)")
	playerBuf := flag.Duration("playerbuf", 0, "oto player buffer (0 = oto default 500ms)")
	prebuf := flag.Duration("prebuf", 2*time.Second, "audio to buffer before Play")
	ringDur := flag.Duration("ring", 60*time.Second, "ring capacity")
	dur := flag.Duration("dur", 30*time.Second, "how long to play")
	vol := flag.Float64("vol", 0, "player volume 0..1 (0 = silent; pipeline still runs)")
	csvPath := flag.String("csv", "", "write per-second telemetry here")
	watch := flag.Bool("watch", true, "poll the player buffer every 5 ms to detect audible gaps")
	prof := flag.String("cpuprofile", "", "write a CPU profile here")
	flag.Parse()
	watchGaps = *watch
	if *prof != "" {
		f, err := os.Create(*prof)
		if err != nil {
			panic(err)
		}
		_ = pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if err := run(*url, *decName, *devBuf, *playerBuf, *prebuf, *ringDur, *dur, *vol, *csvPath); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		pprof.StopCPUProfile()
		os.Exit(1)
	}
}

var watchGaps bool

func run(url, decName string, devBuf, playerBuf, prebuf, ringDur, dur time.Duration, vol float64, csvPath string) error {
	t0 := time.Now()
	since := func() float64 { return time.Since(t0).Seconds() }
	sigCtx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stopSig()
	netCtx, cancelNet := context.WithCancel(context.Background())
	defer cancelNet()

	req, _ := http.NewRequestWithContext(netCtx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Icy-MetaData", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %s", resp.Status)
	}
	metaint, _ := strconv.Atoi(resp.Header.Get("Icy-Metaint"))
	fmt.Printf("connected %.3fs proto=%s type=%q metaint=%d br=%s\n", since(), resp.Proto, resp.Header.Get("Content-Type"), metaint, resp.Header.Get("Icy-Br"))

	var title atomic.Value
	title.Store("")
	body := &counter{r: resp.Body}
	ir := icy.NewReader(body, metaint, func(raw string) { title.Store(icy.Title(raw)) })
	s, err := dec.Open(decName, ir)
	if err != nil {
		return fmt.Errorf("decoder: %w", err)
	}
	rate := s.SampleRate()
	fmt.Printf("first frame %.3fs rate=%d decoder=%s\n", since(), rate, decName)

	octx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate: rate, ChannelCount: 2, Format: oto.FormatSignedInt16LE,
		BufferSize: devBuf, ApplicationName: "mhz-spike",
	})
	if err != nil {
		return fmt.Errorf("oto: %w", err)
	}
	<-ready
	if err := octx.Err(); err != nil {
		return fmt.Errorf("oto init: %w", err)
	}
	fmt.Printf("device ready %.3fs devbuf=%v GOOS=%s\n", since(), devBuf, runtime.GOOS)

	const fb = 4 // bytes per stereo s16 frame
	rb := ring.New(int(ringDur.Seconds()*float64(rate))*fb, fb, time.Now)
	decDone := make(chan error, 1)
	var decFrames atomic.Int64
	go func() {
		buf := make([]byte, 4608)
		for {
			n, err := s.Read(buf)
			n -= n % fb
			rb.Write(buf[:n])
			decFrames.Add(int64(n / fb))
			if err != nil {
				decDone <- err
				return
			}
		}
	}()

	for rb.Stats().Len < int64(prebuf.Seconds()*float64(rate))*fb {
		select {
		case err := <-decDone:
			return fmt.Errorf("stream ended during prebuffer: %w", err)
		case <-time.After(5 * time.Millisecond):
		}
	}
	p := octx.NewPlayer(rb)
	if playerBuf > 0 {
		p.SetBufferSize(int(playerBuf.Seconds()*float64(rate)) * fb)
	}
	p.SetVolume(vol)
	rb.Arm()
	p.Play()
	tPlay := time.Now()
	fmt.Printf("playing %.3fs prebuf=%v playerbuf=%v vol=%.2f\n", since(), prebuf, playerBuf, vol)

	// Watch the player buffer at 5 ms: empty while playing = audible gap.
	var gaps atomic.Int64
	var gapMs atomic.Int64
	stopWatch := make(chan struct{})
	go func() {
		if !watchGaps {
			<-stopWatch
			return
		}
		t := time.NewTicker(5 * time.Millisecond)
		defer t.Stop()
		in := false
		for {
			select {
			case <-stopWatch:
				return
			case <-t.C:
			}
			empty := p.BufferedSize() == 0 && rb.Stats().Len == 0
			if empty {
				gapMs.Add(5)
				if !in {
					gaps.Add(1)
				}
			}
			in = empty
		}
	}()

	var w *csv.Writer
	if csvPath != "" {
		f, err := os.Create(csvPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = csv.NewWriter(f)
		defer w.Flush()
		_ = w.Write([]string{"t_s", "net_bytes", "dec_frames", "played_frames", "ring_ms", "player_ms", "starves", "starved_ms", "gaps", "gap_ms", "dropped", "cpu_ms", "heap_mb", "title"})
	}
	ms := func(bytes int64) float64 { return float64(bytes/fb) * 1000 / float64(rate) }
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	deadline := time.After(dur)
	var endErr error
	lastTitle := ""
loop:
	for {
		select {
		case <-tick.C:
			st := rb.Stats()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			tt := title.Load().(string)
			if tt != lastTitle {
				fmt.Printf("%.0fs title %q\n", time.Since(tPlay).Seconds(), tt)
				lastTitle = tt
			}
			if w != nil {
				_ = w.Write([]string{
					f3(time.Since(tPlay).Seconds()), i64(body.n.Load()), i64(decFrames.Load()), i64(st.Read / fb),
					f3(ms(st.Len)), f3(ms(int64(p.BufferedSize()))), strconv.Itoa(st.Starves), i64(st.Starved.Milliseconds()),
					i64(gaps.Load()), i64(gapMs.Load()), i64(st.Dropped), i64(cpu.Time().Milliseconds()), f3(float64(m.HeapAlloc) / 1e6), tt,
				})
				w.Flush()
			}
		case endErr = <-decDone:
			fmt.Printf("decoder stopped: %v\n", endErr)
			break loop
		case <-deadline:
			break loop
		case <-sigCtx.Done():
			break loop
		}
	}

	// Quit: silence first, then tear down the network side and wait for
	// the decoder goroutine to exit.
	q0 := time.Now()
	p.Pause()
	tPaused := time.Since(q0)
	cancelNet()
	if endErr == nil {
		endErr = <-decDone
	}
	tDec := time.Since(q0)
	close(stopWatch)

	st := rb.Stats()
	played := time.Since(tPlay).Seconds()
	fmt.Printf("summary dec=%s played=%.1fs net=%dB decoded=%d played_frames=%d ring_end=%.0fms starves=%d starved=%v gaps=%d gap_ms=%d dropped=%d cpu=%v\n",
		decName, played, body.n.Load(), decFrames.Load(), st.Read/fb, ms(st.Len), st.Starves, st.Starved, gaps.Load(), gapMs.Load(), st.Dropped, cpu.Time())
	fmt.Printf("consume_rate=%.2f Hz (nominal %d), quit: pause=%v decoder_exit=%v (%v)\n",
		float64(st.Read/fb)/played, rate, tPaused, tDec, endErr)
	return nil
}

func f3(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
func i64(v int64) string  { return strconv.FormatInt(v, 10) }
