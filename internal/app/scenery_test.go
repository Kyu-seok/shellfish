package app_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
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

func TestTheFloorIsTexturedSandThatHoldsStill(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	frames := h.frames(5 * time.Second)
	sand := frames[0][len(frames[0])-2] // just above the status line
	if strings.Trim(sand, sandGlyphs) != "" {
		t.Fatalf("the floor %q is not all sand", sand)
	}
	for _, want := range []string{"░", "▒", "o"} {
		if !strings.Contains(sand, want) {
			t.Errorf("the floor %q has no %q; want textured sand with pebbles", sand, want)
		}
	}
	if strings.Count(sand, "░") <= len([]rune(sand))/2 {
		t.Errorf("the floor %q is not mostly ░", sand)
	}
	for i, rows := range frames {
		if got := rows[len(rows)-2]; got != sand {
			t.Fatalf("frame %d: the floor changed from %q to %q", i, sand, got)
		}
	}
}

func TestTwoShellsRestOnTheSandAndStayPut(t *testing.T) {
	for seed := range uint64(3) {
		h := newHarness(t, named("Bubbles"), withSeed(seed))
		h.resize(80, 24)

		rows := h.view()
		above := floorRow(rows) - 1
		where := map[string]int{}
		for x, c := range cells(rows[above]) {
			if c == "@" || c == "Ѡ" {
				if _, twice := where[c]; twice {
					t.Fatalf("seed %d: more than one %q on the sand:\n%s", seed, c, h.screen())
				}
				where[c] = x
			}
		}
		if len(where) != 2 {
			t.Fatalf("seed %d: shells on the sand: %v, want one @ and one Ѡ:\n%s", seed, where, h.screen())
		}
		for i := range int(20 * time.Second / app.FrameInterval) {
			h.advance(app.FrameInterval)
			rows := h.view()
			fish, seen := findFish(rows)
			for shell, x := range where {
				hidden := seen && x >= fish.left && x < fish.left+fishBoxWidth && above >= fish.top && above < fish.top+3
				if got := cells(rows[above])[x]; got != shell && !hidden {
					t.Fatalf("seed %d frame %d: the %q shell at column %d is gone:\n%s", seed, i, shell, x, strings.Join(rows, "\n"))
				}
			}
		}
	}
}
