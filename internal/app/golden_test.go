package app

import (
	"path/filepath"
	"testing"

	"github.com/jacob-lineberry/mhz/internal/golden"
)

func TestPlaceholderGolden(t *testing.T) {
	sizes := []struct {
		name string
		w, h int
	}{
		{"normal", 80, 24},
		{"large", 120, 40},
		{"exact fit", 15, 3},
		{"tiny", 10, 2},
	}
	for _, s := range sizes {
		t.Run(s.name, func(t *testing.T) {
			golden.RequireView(t, New(), s.w, s.h)
		})
	}
}

func TestPlaceholderGoldenUnsized(t *testing.T) {
	golden.RequireEqual(t,
		filepath.Join("testdata", "TestPlaceholderGoldenUnsized", "unsized.golden"),
		[]byte(New().View().Content))
}
