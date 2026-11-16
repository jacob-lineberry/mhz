// Package icy strips in-band ICY metadata from a SHOUTcast/Icecast body.
//
// Wire format: metaint audio bytes, one length byte L, then L*16 bytes of
// NUL-padded text such as "StreamTitle='Artist - Title';", repeated.
package icy

import (
	"bytes"
	"io"
)

// Reader yields only audio bytes. OnMeta is called with the raw text of
// each non-empty metadata block, on the reading goroutine.
type Reader struct {
	r         io.Reader
	metaint   int
	remaining int
	OnMeta    func(raw string)
	Blocks    int // metadata blocks seen, including empty ones
	buf       [255 * 16]byte
}

// NewReader returns a Reader. metaint <= 0 means the stream has no
// metadata and r is passed through.
func NewReader(r io.Reader, metaint int, onMeta func(string)) *Reader {
	return &Reader{r: r, metaint: metaint, remaining: metaint, OnMeta: onMeta}
}

func (ir *Reader) Read(p []byte) (int, error) {
	if ir.metaint <= 0 {
		return ir.r.Read(p)
	}
	if ir.remaining == 0 {
		if err := ir.meta(); err != nil {
			return 0, err
		}
		ir.remaining = ir.metaint
	}
	if len(p) > ir.remaining {
		p = p[:ir.remaining]
	}
	n, err := ir.r.Read(p)
	ir.remaining -= n
	return n, err
}

func (ir *Reader) meta() error {
	var l [1]byte
	if _, err := io.ReadFull(ir.r, l[:]); err != nil {
		return err
	}
	ir.Blocks++
	n := int(l[0]) * 16
	if n == 0 {
		return nil
	}
	b := ir.buf[:n]
	if _, err := io.ReadFull(ir.r, b); err != nil {
		return err
	}
	if ir.OnMeta != nil {
		ir.OnMeta(string(bytes.TrimRight(b, "\x00")))
	}
	return nil
}

// Title extracts StreamTitle from a raw block. It returns "" if absent.
func Title(raw string) string {
	const key = "StreamTitle='"
	i := bytes.Index([]byte(raw), []byte(key))
	if i < 0 {
		return ""
	}
	rest := raw[i+len(key):]
	if j := bytes.Index([]byte(rest), []byte("';")); j >= 0 {
		return rest[:j]
	}
	if j := bytes.LastIndexByte([]byte(rest), '\''); j >= 0 {
		return rest[:j]
	}
	return rest
}
