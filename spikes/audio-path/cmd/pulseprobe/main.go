//go:build linux

// Command pulseprobe asks PulseAudio for a range of latencies, the same way
// oto's Pulse driver does (jfreymuth/pulse, float32 callback), and reports
// whether the server saw an underflow. oto does not expose the stream, so
// this is the only way to see device-level underruns on Linux.
//
//	pulseprobe -lat 5ms,10ms,20ms,50ms,100ms -each 60s [-load]
package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/jfreymuth/pulse"
)

var sink []byte

func main() {
	lats := flag.String("lat", "5ms,10ms,20ms,50ms,100ms", "latencies to try")
	each := flag.Duration("each", 60*time.Second, "time per latency")
	load := flag.Bool("load", false, "allocate heavily in the background (GC pressure)")
	flag.Parse()
	if *load {
		go func() {
			for {
				sink = make([]byte, 1<<20)
			}
		}()
	}
	for _, s := range strings.Split(*lats, ",") {
		lat, err := time.ParseDuration(s)
		if err != nil {
			panic(err)
		}
		probe(lat, *each)
	}
}

func probe(lat, each time.Duration) {
	c, err := pulse.NewClient(pulse.ClientApplicationName("mhz-spike-probe"))
	if err != nil {
		fmt.Println("client:", err)
		return
	}
	defer c.Close()
	calls := 0
	var maxGap, last time.Duration
	t0 := time.Now()
	read := func(buf []float32) (int, error) {
		now := time.Since(t0)
		if calls > 0 {
			maxGap = max(maxGap, now-last)
		}
		last = now
		calls++
		clear(buf) // silence: underflow detection does not need sound
		return len(buf), nil
	}
	st, err := c.NewPlayback(pulse.Float32Reader(read), pulse.PlaybackStereo, pulse.PlaybackSampleRate(44100), pulse.PlaybackLatency(lat.Seconds()))
	if err != nil {
		fmt.Println("playback:", err)
		return
	}
	st.Start()
	first := time.Duration(-1)
	end := time.Now().Add(each)
	for time.Now().Before(end) {
		time.Sleep(20 * time.Millisecond)
		if first < 0 && st.Underflow() {
			first = time.Since(t0)
		}
	}
	st.Stop()
	st.Close()
	res := "no underflow"
	if first >= 0 {
		res = fmt.Sprintf("UNDERFLOW first at %v", first.Round(time.Millisecond))
	}
	fmt.Printf("latency=%-6v granted_buf=%d frames (%.1f ms) callbacks=%d max_gap=%v  %s\n",
		lat, st.BufferSize(), float64(st.BufferSize())*1000/44100, calls, maxGap.Round(time.Millisecond), res)
}
