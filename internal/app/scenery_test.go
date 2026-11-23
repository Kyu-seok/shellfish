package app_test

import (
	"strings"
	"testing"
	"time"
)

// isWave reports whether row is the water's surface: a whole row of wave.
func isWave(row string) bool {
	return row != "" && strings.Trim(row, "~-_") == ""
}

func TestTheSurfaceIsADriftingWaveLine(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	frames := h.frames(5 * time.Second)
	changed := false
	for i, rows := range frames {
		if !isWave(rows[0]) {
			t.Fatalf("frame %d: the top row is not a wave line: %q", i, rows[0])
		}
		changed = changed || rows[0] != frames[0][0]
	}
	if !changed {
		t.Errorf("the surface stayed %q for 5 seconds; want it drifting", frames[0][0])
	}
}

// Food dropped at the top, bubbles rising and hearts and notes floating up
// all stop below the surface, as does the Fish.
func TestNothingDrawsOnTheSurface(t *testing.T) {
	for seed := range uint64(3) {
		h := inMood(t, "Content", withSeed(seed))
		h.resize(80, 24)
		for i := range 40 {
			if i%10 == 0 {
				h.press("f")
				h.click(10+i, 0)
			}
			for _, rows := range h.frames(time.Second) {
				if !isWave(rows[0]) {
					t.Fatalf("seed %d: something is drawn on the surface:\n%s", seed, strings.Join(rows, "\n"))
				}
				if s, ok := findFish(rows); ok && s.top < 1 {
					t.Fatalf("seed %d: the Fish swims at the surface:\n%s", seed, strings.Join(rows, "\n"))
				}
			}
		}
	}
}
