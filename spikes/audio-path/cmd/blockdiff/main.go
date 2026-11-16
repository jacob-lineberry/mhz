// Command blockdiff lists the 1152-frame blocks where two PCM files differ
// most, with each file's block energy next to its neighbours' energy. A
// decoder glitch shows as energy far from both neighbours in one file only.
//
//	blockdiff -a tphakala.s16 -b hajimehoshi.s16 -off 1152 -n 15
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
)

func load(p string) []int16 {
	b, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}
	s := make([]int16, len(b)/2)
	for i := range s {
		s[i] = int16(binary.LittleEndian.Uint16(b[2*i:]))
	}
	return s
}

func main() {
	ap, bp := flag.String("a", "", "file A"), flag.String("b", "", "file B")
	off := flag.Int("off", 0, "B is ahead of A by this many frames")
	n := flag.Int("n", 15, "blocks to list")
	flag.Parse()
	a, b := load(*ap), load(*bp)
	const blk = 1152 * 2
	nb := min(len(a), len(b)-2**off) / blk
	ea, eb := make([]float64, nb), make([]float64, nb)
	type d struct {
		i    int
		snr  float64
		peak float64
	}
	var ds []d
	for k := range nb {
		var sa, sb, sn, pk float64
		for i := k * blk; i < (k+1)*blk; i++ {
			x, y := float64(a[i]), float64(b[i+2**off])
			sa += x * x
			sb += y * y
			sn += (x - y) * (x - y)
			pk = max(pk, math.Abs(x-y))
		}
		ea[k], eb[k] = 10*math.Log10(sa/blk+1), 10*math.Log10(sb/blk+1)
		ds = append(ds, d{k, 10 * math.Log10((sa+1)/(sn+1)), pk})
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].snr < ds[j].snr })
	fmt.Println("block  time_s  snr_dB  peak_diff  A_dB(prev,this,next)  B_dB(prev,this,next)")
	for _, x := range ds[:min(*n, len(ds))] {
		k := x.i
		g := func(e []float64, i int) float64 {
			if i < 0 || i >= len(e) {
				return math.NaN()
			}
			return e[i]
		}
		fmt.Printf("%5d %7.2f %7.1f %9.0f   %5.1f %5.1f %5.1f      %5.1f %5.1f %5.1f\n", k, float64(k*1152)/44100, x.snr, x.peak,
			g(ea, k-1), ea[k], g(ea, k+1), g(eb, k-1), eb[k], g(eb, k+1))
	}
}
