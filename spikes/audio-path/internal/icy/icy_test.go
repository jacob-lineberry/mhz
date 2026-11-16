package icy

import (
	"bytes"
	"io"
	"testing"
	"testing/iotest"
)

func build(metaint int, audio []byte, metas []string) []byte {
	var out bytes.Buffer
	for i := 0; len(audio) > 0; i++ {
		n := min(metaint, len(audio))
		out.Write(audio[:n])
		audio = audio[n:]
		if n < metaint {
			break
		}
		m := ""
		if i < len(metas) {
			m = metas[i]
		}
		l := (len(m) + 15) / 16
		out.WriteByte(byte(l))
		out.WriteString(m)
		out.Write(make([]byte, l*16-len(m)))
	}
	return out.Bytes()
}

func TestStripsMetadataAcrossOneByteReads(t *testing.T) {
	audio := make([]byte, 1000)
	for i := range audio {
		audio[i] = byte(i)
	}
	wire := build(100, audio, []string{"StreamTitle='A - B';", "", "StreamTitle='It''s';StreamUrl='x';"})
	var got []string
	r := NewReader(iotest.OneByteReader(bytes.NewReader(wire)), 100, func(s string) { got = append(got, Title(s)) })
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, audio) {
		t.Fatalf("audio mismatch: got %d bytes", len(out))
	}
	if len(got) != 2 || got[0] != "A - B" || got[1] != "It''s" {
		t.Fatalf("titles = %q", got)
	}
	if r.Blocks != 10 {
		t.Fatalf("blocks = %d", r.Blocks)
	}
}
