package app_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
	"github.com/Kyu-seok/shellfish/internal/fish"
	"github.com/Kyu-seok/shellfish/internal/tank"
)

// Every Species swims, eats and gets hungry the same way, and shows how it
// is in the same way: these tests run each check on all ten.

// eachSpecies runs check as a subtest for every Species, side by side.
func eachSpecies(t *testing.T, check func(t *testing.T, sp fish.Species)) {
	for _, sp := range fish.AllSpecies {
		t.Run(sp.String(), func(t *testing.T) {
			t.Parallel()
			check(t, sp)
		})
	}
}

func TestEverySpeciesEyeShowsItsMoodAndBlinks(t *testing.T) {
	eachSpecies(t, func(t *testing.T, sp fish.Species) {
		for _, mood := range []string{"Content", "Hungry", "Starving"} {
			h := inMood(t, mood, as(sp))
			h.resize(80, 24)
			h.advance(app.GreetingDuration) // the greeting may cover the Fish

			blinks := 0
			for i, s := range h.watch(8 * time.Second) {
				switch {
				case s.blinking:
					blinks++
				case s.mood != mood:
					t.Fatalf("%s, frame %d: the eye looks %q:\n%s", mood, i, s.mood, h.screen())
				}
			}
			if blinks == 0 {
				t.Errorf("%s: the Fish never blinked in 8 seconds", mood)
			}
		}
	})
}

func TestEverySpeciesOpensItsMouthAndEatsWithIt(t *testing.T) {
	eachSpecies(t, func(t *testing.T, sp fish.Species) {
		h := hungerAfter(t, 12*time.Hour, as(sp)) // Hungry
		s, ok := h.find(h.rawView())
		if !ok {
			t.Fatalf("no Fish in the Tank:\n%s", h.screen())
		}
		// Food resting across the Tank from the Fish, so it must swim there.
		col := 5
		if s.left < 40 {
			col = 74
		}
		h.click(col, h.bottom())
		row := -1
		for y := range h.bottom() + 1 {
			if h.cellAt(col, y) == pellet {
				row = y // on the sand, or on top of a stalk
			}
		}
		if row < 0 {
			t.Fatalf("the pellet dropped at column %d is not to be seen:\n%s", col, h.screen())
		}

		before := readSave(t, h.dataDir).Hunger
		var open []bool // whether its mouth was open, each frame
		for len(open) < int(30*time.Second/app.FrameInterval) && readSave(t, h.dataDir).Hunger >= before {
			h.advance(app.FrameInterval)
			s, _ = h.find(h.rawView())
			open = append(open, s.eating)
		}
		if readSave(t, h.dataDir).Hunger >= before {
			t.Fatalf("the Fish ate nothing in 30 seconds:\n%s", h.screen())
		}
		// It ate with its mouth: the Food was right by it.
		b := h.body()
		mouthCol, mouthRow := s.left+b.MouthRight, s.top+b.MouthRow
		if s.facingLeft {
			mouthCol = s.left + b.MouthLeft
		}
		if abs(mouthCol-col) > 2 || abs(mouthRow-row) > 2 {
			t.Errorf("the Fish ate the Food at (%d, %d) with its mouth at (%d, %d):\n%s", col, row, mouthCol, mouthRow, h.screen())
		}
		last := -1
		for i, o := range open {
			if o {
				last = i
			}
		}
		switch {
		case last < 0:
			t.Error("the Fish never opened its mouth on its way to the Food")
		case len(open)-last > 3:
			t.Errorf("the Fish last had its mouth open %d frames before it ate, want just before", len(open)-last)
		}
		if open[0] {
			t.Error("the Fish opened its mouth as soon as the Food was dropped")
		}
	})
}

func TestEverySpeciesSwimsWholeFacingAheadAndOverTheTallestStalk(t *testing.T) {
	eachSpecies(t, func(t *testing.T, sp fish.Species) {
		for seed := range uint64(2) {
			h := newHarness(t, as(sp), withSeed(seed), named("Bubbles"))
			h.resize(80, 24)
			// The tallest stalk, by the column its top is drawn in.
			tallest := plant{row: 1 << 30}
			for _, p := range h.plants() {
				if p.row < tallest.row {
					tallest = p
				}
			}
			if tallest.row == 1<<30 {
				t.Fatalf("seed %d: no seaweed to swim over:\n%s", seed, h.screen())
			}

			// watch fails the test if the Fish is ever cut off, outside the
			// Tank, on the floor or in front of seaweed.
			seen := h.watch(2 * time.Minute)
			leftOf, rightOf := false, false
			for i, s := range seen {
				leftOf = leftOf || s.left+h.body().Width < tallest.col-1
				rightOf = rightOf || s.left > tallest.col+1
				if i == 0 || s.facingLeft != seen[i-1].facingLeft {
					continue // turning around
				}
				if dx := s.left - seen[i-1].left; dx < 0 && !s.facingLeft || dx > 0 && s.facingLeft {
					t.Fatalf("seed %d frame %d: the Fish swam %d columns facing the other way", seed, i, dx)
				}
			}
			if !leftOf || !rightOf {
				t.Errorf("seed %d: in two minutes the Fish was never on both sides of the tallest stalk, at column %d", seed, tallest.col)
			}
		}
	})
}

func TestEverySpeciesStaysInsideTheTankAsItIsResized(t *testing.T) {
	eachSpecies(t, func(t *testing.T, sp fish.Species) {
		h := newHarness(t, as(sp), named("Bubbles"))
		h.resize(160, 50)
		sizes := []struct{ w, h int }{{30, 10}, {120, 40}, {40, 12}, {31, 11}, {80, 24}, {30, 10}}
		for round := range 12 {
			size := sizes[round%len(sizes)]
			h.resize(size.w, size.h)
			// watch fails the test if the Fish is ever cut off, outside
			// the Tank, on the floor or in front of seaweed.
			h.watch(time.Duration(1+round%3) * 2 * time.Second)
		}
	})
}

// The Fish can reach Food resting anywhere on the floor, however wide it
// is: the seaweed leaves it room. The Tanks are the tightest the seaweed
// grows in: one stalk with the fewest columns it allows on either side, and
// one stalk against a wall.
func TestEverySpeciesReachesFoodOnEveryColumnOfTheFloor(t *testing.T) {
	sizes := []struct{ w, h int }{
		{2*tank.SeaweedGap + tank.SeaweedWidth, 16},
		{tank.SeaweedGap + tank.SeaweedWidth + 3, 12},
	}
	eachSpecies(t, func(t *testing.T, sp fish.Species) {
		for seed, size := range sizes {
			h := hungerAfter(t, 12*time.Hour, as(sp), withSeed(uint64(seed))) // Hunger 50
			h.resize(size.w, size.h)
			if seaweedCells(h.view()) == 0 {
				t.Fatalf("%dx%d: no seaweed to get in the way:\n%s", size.w, size.h, h.screen())
			}
			for col := range size.w {
				h.clock.now = h.clock.now.Add(144 * time.Minute) // Hunger up 10, as much as a pellet takes off
				h.advance(app.FrameInterval)                     // which the periodic save notes
				h.click(col, h.bottom())
				if !h.eats(10 * time.Second) {
					t.Fatalf("%dx%d: the Fish never ate the Food dropped on the floor at column %d:\n%s",
						size.w, size.h, col, h.screen())
				}
			}
		}
	})
}

// eats advances up to limit, a frame at a time, until the Fish eats, and
// reports whether it did.
func (h *harness) eats(limit time.Duration) bool {
	before := readSave(h.t, h.dataDir).Hunger
	for range int(limit / app.FrameInterval) {
		h.advance(app.FrameInterval)
		if readSave(h.t, h.dataDir).Hunger < before {
			return true
		}
	}
	return false
}

// In monochrome every Species is a plain silhouette with its eye a hole
// in it, closed all round, facing either way.
func TestEverySpeciesHasAHoleForItsEyeInMonochrome(t *testing.T) {
	eachSpecies(t, func(t *testing.T, sp fish.Species) {
		h := newHarness(t, as(sp), named("Bubbles"), monochrome())
		h.resize(80, 24)
		turned := map[bool]bool{}
		for i := range int(30 * time.Second / app.FrameInterval) {
			h.advance(app.FrameInterval)
			rows := h.view()
			s, ok := h.find(h.rawView())
			if !ok || !s.intact {
				t.Fatalf("frame %d: no whole Fish found in monochrome:\n%s", i, strings.Join(rows, "\n"))
			}
			if !eyeHole(rows, sp.Art(), s.left, s.top, s.facingLeft) {
				t.Fatalf("frame %d: the Fish's eye is not a hole in its silhouette:\n%s", i, strings.Join(rows, "\n"))
			}
			turned[s.facingLeft] = true
		}
		if len(turned) < 2 {
			t.Error("the Fish never turned in 30 seconds, so only one way was checked")
		}
	})
}
