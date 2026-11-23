package app_test

import (
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

func TestTheFishIsAPixelClownfish(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	s, ok := findFish(h.rawView())
	if !ok || !s.intact {
		t.Fatalf("no whole Fish in the Tank:\n%s", h.screen())
	}
	// Everything in its box is drawn in pixels or left clear: no text.
	rows := h.view()
	for y := s.top; y < s.top+body.Height; y++ {
		row := cells(rows[y])
		for x := s.left; x < s.left+body.Width; x++ {
			if c := row[x]; c != " " && !isFishCell(c) {
				t.Errorf("cell (%d, %d) of the Fish is %q, not a pixel:\n%s", x, y, c, h.screen())
			}
		}
	}
	// A clownfish is orange with white bands.
	seen := map[string]bool{}
	for _, c := range fishCells(rows) {
		st := h.styleAt(c[0], c[1])
		seen[st.fg], seen[st.bg] = true, true
	}
	for _, want := range []string{"#ff8a1f", "#ffffff"} {
		if !hasColour(seen, want) {
			t.Errorf("the Fish has no %s in it; its colours are %v", want, seen)
		}
	}
}

func TestTheFishsOutlineLeavesTheTerminalsBackgroundShowing(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	for frame := range 40 {
		h.advance(app.FrameInterval)
		raw := h.rawView()
		s, ok := findFish(raw)
		if !ok || !s.intact {
			t.Fatalf("frame %d: no whole Fish in the Tank:\n%s", frame, h.screen())
		}
		for y := s.top; y < s.top+body.Height; y++ {
			var ends []int
			for x := s.left; x < s.left+body.Width; x++ {
				if !isFishCell(cells(h.view()[y])[x]) {
					continue
				}
				if st := styleIn(raw[y], x); st.fg == "" {
					t.Fatalf("frame %d: Fish cell (%d, %d) has no colour: %q", frame, x, y, raw[y])
				}
				ends = append(ends, x)
			}
			// The front of each row is the rounded head and its fins,
			// with water showing around them: the cell leaves the
			// terminal's background as it is.
			x := ends[len(ends)-1]
			if s.facingLeft {
				x = ends[0]
			}
			if st := styleIn(raw[y], x); st.bg != "" {
				t.Errorf("frame %d: the front cell (%d, %d) of the Fish paints the background %s: %q", frame, x, y, st.bg, raw[y])
			}
		}
	}
}

func TestTheFishFacesTheWayItSwims(t *testing.T) {
	for seed := range uint64(3) {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(80, 24)
		seen := h.watch(time.Minute)

		swamLeft, swamRight := 0, 0
		for i := 1; i < len(seen); i++ {
			if seen[i].facingLeft != seen[i-1].facingLeft {
				continue // turning around
			}
			switch dx := seen[i].left - seen[i-1].left; {
			case dx < 0 && !seen[i].facingLeft, dx > 0 && seen[i].facingLeft:
				t.Fatalf("seed %d frame %d: the Fish swam %d columns with its eye facing the other way", seed, i, dx)
			case dx < 0:
				swamLeft++
			case dx > 0:
				swamRight++
			}
		}
		if swamLeft == 0 || swamRight == 0 {
			t.Errorf("seed %d: in a minute the Fish swam left %d times and right %d times; want both", seed, swamLeft, swamRight)
		}
	}
}
