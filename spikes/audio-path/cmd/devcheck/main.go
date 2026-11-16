// Command devcheck opens an oto context and reports how long it took and
// what error, if any, came back. Used to see what a machine with no audio
// device (CI) does.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ebitengine/oto/v3"
)

func main() {
	t0 := time.Now()
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{SampleRate: 44100, ChannelCount: 2, Format: oto.FormatSignedInt16LE})
	if err != nil {
		fmt.Printf("NewContext error after %v: %v\n", time.Since(t0), err)
		os.Exit(1)
	}
	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		fmt.Println("ready did not close within 10s")
		os.Exit(1)
	}
	if err := ctx.Err(); err != nil {
		fmt.Printf("init error after %v: %v\n", time.Since(t0), err)
		os.Exit(1)
	}
	fmt.Printf("device ready after %v\n", time.Since(t0))
}
