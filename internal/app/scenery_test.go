package app_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
	"github.com/Kyu-seok/shellfish/internal/tank"
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
				if _, top, _, _, ok := fishBounds(rows); ok && top < 1 {
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

func TestTwoShellsRestOnTheSandClearOfTheSeaweedAndStayPut(t *testing.T) {
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
			fish, seen := findFish(h.rawView())
			for _, p := range h.plants() {
				if seen && p.col >= fish.left && p.col < fish.left+body.Width && p.row >= fish.top && p.row < fish.top+body.Height {
					continue // the Fish, not seaweed
				}
				for shell, x := range where {
					if d := abs(p.col - x); d <= tank.ShellClearance {
						t.Fatalf("seed %d frame %d: seaweed %q at column %d stands %d from the %q shell at %d, want more than %d:\n%s",
							seed, i, p.glyph, p.col, d, shell, x, tank.ShellClearance, strings.Join(rows, "\n"))
					}
				}
			}
			for shell, x := range where {
				hidden := seen && x >= fish.left && x < fish.left+body.Width && above >= fish.top && above < fish.top+body.Height
				if got := cells(rows[above])[x]; got != shell && !hidden {
					t.Fatalf("seed %d frame %d: the %q shell at column %d is gone:\n%s", seed, i, shell, x, strings.Join(rows, "\n"))
				}
			}
		}
	}
}

// The Scenery's colours, by what they look like rather than exact shades.
var (
	blue = func(r, g, b int) bool { return b > r+40 && b > g }
	tan  = func(r, g, b int) bool { return r > g && g > b && r-b > 40 }
	grey = func(r, g, b int) bool { return max(r, g, b)-min(r, g, b) < 16 }
	pink = func(r, g, b int) bool { return r > 200 && r > g+40 && r > b+40 }
	// paleCyan is light, with green and blue above red.
	paleCyan = func(r, g, b int) bool { return min(r, g, b) > 150 && g > r && b > r }
)

// colouredAs reports whether st's foreground is one of looks.
func colouredAs(st style, looks ...func(r, g, b int) bool) bool {
	r, g, b, ok := rgbOf(st.fg)
	if !ok {
		return false
	}
	for _, l := range looks {
		if l(r, g, b) {
			return true
		}
	}
	return false
}

func TestTheSceneryIsInColourWithNoBackground(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	bubble := false
	for range 10 {
		h.advance(time.Second)
		rows := h.view()
		floor := floorRow(rows)
		fish, seen := findFish(h.rawView())
		for y, row := range rows {
			for x, c := range cells(row) {
				st := h.styleAt(x, y)
				inFish := seen && x >= fish.left && x < fish.left+body.Width && y >= fish.top && y < fish.top+body.Height
				if st.bg != "" && !inFish {
					t.Fatalf("cell (%d, %d) %q has a background colour %s", x, y, c, st.bg)
				}
				switch {
				case y == 0 && !colouredAs(st, blue):
					t.Fatalf("the surface's %q at column %d is %q, want blue", c, x, st.fg)
				case y == floor && c == "o" && !colouredAs(st, grey):
					t.Fatalf("the pebble at column %d is %q, want grey", x, st.fg)
				case y == floor && c != "o" && !colouredAs(st, tan):
					t.Fatalf("the sand's %q at column %d is %q, want tan", c, x, st.fg)
				case y == floor-1 && (c == "@" || c == "Ѡ") && !colouredAs(st, pink):
					t.Fatalf("the shell %q at column %d is %q, want pink", c, x, st.fg)
				case y > 0 && y < floor && !inFish && strings.Contains("°oO", c) && c != "":
					bubble = true
					if !colouredAs(st, paleCyan) {
						t.Fatalf("the bubble %q at (%d, %d) is %q, want pale cyan", c, x, y, st.fg)
					}
				}
			}
		}
	}
	if !bubble {
		t.Error("no bubble rose in 10 seconds to check the colour of")
	}
}
