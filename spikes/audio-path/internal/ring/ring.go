// Package ring is the PCM buffer between the decoder goroutine and the
// audio device. Read never blocks: when empty it returns 0, nil, which oto's
// mux treats as "nothing yet" (it sleeps 1 ms and retries). A blocking Read
// would stall oto's mux goroutine and make Player.Pause wait on the network.
package ring

import (
	"sync"
	"time"
)

// Ring is a fixed-capacity byte FIFO safe for one writer and one reader.
type Ring struct {
	mu       sync.Mutex
	buf      []byte
	r, n     int
	align    int
	written  int64
	read     int64
	dropped  int64
	starving bool
	starveAt time.Time
	starves  int
	starved  time.Duration
	now      func() time.Time
	armed    bool
}

// New returns a ring holding size bytes; reads and writes stay multiples of
// align (bytes per sample frame).
func New(size, align int, now func() time.Time) *Ring {
	size -= size % align
	return &Ring{buf: make([]byte, size), align: align, now: now}
}

// Arm starts counting starvation (call when playback starts).
func (b *Ring) Arm() { b.mu.Lock(); b.armed = true; b.mu.Unlock() }

// Write copies p in, dropping whatever does not fit. It never blocks.
func (b *Ring) Write(p []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	free := len(b.buf) - b.n
	k := min(len(p), free)
	k -= k % b.align
	for i := 0; i < k; {
		w := (b.r + b.n) % len(b.buf)
		c := copy(b.buf[w:min(len(b.buf), w+k-i)], p[i:k])
		b.n += c
		i += c
	}
	b.written += int64(k)
	b.dropped += int64(len(p) - k)
}

// Read implements io.Reader for oto.
func (b *Ring) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	k := min(len(p), b.n)
	k -= k % b.align
	if k == 0 {
		if b.armed && !b.starving {
			b.starving, b.starveAt = true, b.now()
			b.starves++
		}
		return 0, nil
	}
	if b.starving {
		b.starving = false
		b.starved += b.now().Sub(b.starveAt)
	}
	for i := 0; i < k; {
		c := copy(p[i:k], b.buf[b.r:min(len(b.buf), b.r+k-i)])
		b.r = (b.r + c) % len(b.buf)
		b.n -= c
		i += c
	}
	b.read += int64(k)
	return k, nil
}

// Stats is a snapshot of counters.
type Stats struct {
	Len, Written, Read, Dropped int64
	Starves                     int
	Starved                     time.Duration
}

// Stats returns a snapshot.
func (b *Ring) Stats() Stats {
	b.mu.Lock()
	defer b.mu.Unlock()
	s := Stats{Len: int64(b.n), Written: b.written, Read: b.read, Dropped: b.dropped, Starves: b.starves, Starved: b.starved}
	if b.starving {
		s.Starved += b.now().Sub(b.starveAt)
	}
	return s
}
