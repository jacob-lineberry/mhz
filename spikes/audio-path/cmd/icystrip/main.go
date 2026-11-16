// Command icystrip removes ICY metadata from a captured stream body and
// prints each metadata block.
//
//	icystrip -metaint 16000 < in.bin > out.mp3
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/jacob-lineberry/mhz/spikes/audio-path/internal/icy"
)

func main() {
	metaint := flag.Int("metaint", 16000, "icy-metaint from the response headers")
	flag.Parse()
	r := icy.NewReader(os.Stdin, *metaint, func(raw string) { fmt.Fprintf(os.Stderr, "meta: %q\n", raw) })
	n, err := io.Copy(os.Stdout, r)
	fmt.Fprintf(os.Stderr, "audio bytes=%d blocks=%d err=%v\n", n, r.Blocks, err)
}
