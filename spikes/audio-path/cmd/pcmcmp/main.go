// Command pcmcmp compares two s16le stereo PCM files block by block.
// It finds the alignment offset first, then reports overall SNR and the
// number of 1152-sample blocks below a threshold (audible differences).
//
//	pcmcmp -ref tphakala.s16 -test hajimehoshi.s16
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
)

func load(p string) []int16 {
	b, err := os.ReadFile(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := make([]int16, len(b)/2)
	for i := range s {
		s[i] = int16(binary.LittleEndian.Uint16(b[2*i:]))
	}
	return s
}

func main() {
	refP := flag.String("ref", "", "reference PCM")
	testP := flag.String("test", "", "PCM to compare")
	thr := flag.Float64("thr", 40, "block SNR threshold in dB")
	flag.Parse()
	ref, test := load(*refP), load(*testP)

	// Alignment: try offsets of whole stereo frames in [-4608, 4608] on a
	// 2 s window starting 5 s in (past any start-up transient).
	best, bestErr := 0, math.Inf(1)
	start, win := 5*44100*2, 2*44100*2
	for off := -4608; off <= 4608; off++ {
		var e float64
		for i := start; i < start+win; i += 2 {
			j := i + 2*off
			if j < 0 || j+1 >= len(test) || i+1 >= len(ref) {
				e = math.Inf(1)
				break
			}
			d := float64(ref[i]) - float64(test[j])
			e += d * d
		}
		if e < bestErr {
			best, bestErr = off, e
		}
	}

	const block = 1152 * 2
	var sig, noise float64
	bad, blocks, maxAbs := 0, 0, 0.0
	for b := 0; ; b++ {
		i0 := b * block
		if i0+block > len(ref) {
			break
		}
		var bs, bn float64
		ok := true
		for i := i0; i < i0+block; i++ {
			j := i + 2*best
			if j < 0 || j >= len(test) {
				ok = false
				break
			}
			r, t := float64(ref[i]), float64(test[j])
			bs += r * r
			bn += (r - t) * (r - t)
			maxAbs = max(maxAbs, math.Abs(r-t))
		}
		if !ok {
			break
		}
		blocks++
		sig += bs
		noise += bn
		if bs > 1e3*block && 10*math.Log10(bs/max(bn, 1e-9)) < *thr {
			bad++
		}
	}
	fmt.Printf("ref=%d test=%d frames, offset=%d, blocks=%d, SNR=%.1f dB, blocks<%.0fdB=%d, max|diff|=%.0f\n",
		len(ref)/2, len(test)/2, best, blocks, 10*math.Log10(sig/max(noise, 1e-9)), *thr, bad, maxAbs)
}
