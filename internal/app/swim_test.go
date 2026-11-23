package app_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

func TestFishWandersAroundTheTankWithoutLeavingIt(t *testing.T) {
	for seed := range uint64(5) {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(80, 24)
		seen := h.watch(2 * time.Minute)

		minCol, maxCol, minRow, maxRow := 80, 0, 24, 0
		for _, s := range seen {
			minCol, maxCol = min(minCol, s.left), max(maxCol, s.left)
			minRow, maxRow = min(minRow, s.top), max(maxRow, s.top)
		}
		if maxCol-minCol < 35 {
			t.Errorf("seed %d: the Fish only roamed columns %d–%d", seed, minCol, maxCol)
		}
		if maxRow-minRow < 6 {
			t.Errorf("seed %d: the Fish only roamed rows %d–%d", seed, minRow, maxRow)
		}
	}
}

func TestFishHoldsStillBeforeTurningAround(t *testing.T) {
	const hold = 3 // frames the Fish must stay in one column before it mirrors
	for seed := range uint64(5) {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(80, 24)
		seen := h.watch(2 * time.Minute)

		turns := 0
		for i := hold; i < len(seen); i++ {
			if seen[i].facingLeft == seen[i-1].facingLeft {
				continue
			}
			turns++
			for j := i - hold; j < i; j++ {
				if seen[j].left != seen[i].left {
					t.Errorf("seed %d frame %d: the Fish turned around while moving (columns %d then %d)",
						seed, i, seen[j].left, seen[i].left)
					break
				}
			}
		}
		if turns < 3 {
			t.Errorf("seed %d: the Fish turned around only %d times in two minutes", seed, turns)
		}
	}
}

// stillStretches returns each run of frames in which the Fish stayed in one
// column, as [first, last] frame indexes.
func stillStretches(seen []sighting) [][2]int {
	var runs [][2]int
	start := 0
	for i := 1; i <= len(seen); i++ {
		if i == len(seen) || seen[i].left != seen[start].left {
			runs = append(runs, [2]int{start, i - 1})
			start = i
		}
	}
	return runs
}

func TestFishPausesAndBobsBetweenTrips(t *testing.T) {
	second := int(time.Second / app.FrameInterval)
	for seed := range uint64(5) {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(80, 24)
		seen := h.watch(2 * time.Minute)

		pauses := 0
		for _, run := range stillStretches(seen) {
			if run[1]-run[0]+1 < 3*second {
				continue
			}
			pauses++
			rose, sank := false, false
			for i := run[0] + 1; i <= run[1]; i++ {
				rose = rose || seen[i].top < seen[i-1].top
				sank = sank || seen[i].top > seen[i-1].top
			}
			if !rose || !sank {
				t.Errorf("seed %d: the Fish hovered at frame %d for %d frames without bobbing up and down",
					seed, run[0], run[1]-run[0]+1)
			}
		}
		if pauses < 5 {
			t.Errorf("seed %d: the Fish paused for 3 seconds or more only %d times in two minutes", seed, pauses)
		}
	}
}

func TestFishGlidesSmoothlyAndSometimesDarts(t *testing.T) {
	half := int(time.Second / app.FrameInterval / 2)
	for seed := range uint64(5) {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(80, 24)
		seen := h.watch(3 * time.Minute)

		for i := 1; i < len(seen); i++ {
			if dx, dy := abs(seen[i].left-seen[i-1].left), abs(seen[i].top-seen[i-1].top); dx > 1 || dy > 1 {
				t.Fatalf("seed %d frame %d: the Fish jumped %d columns and %d rows in one frame", seed, i, dx, dy)
			}
		}
		// Cruising, the Fish covers at most 4 columns in half a second. A
		// dart covers far more.
		darts := 0
		for i := half; i < len(seen); i++ {
			if abs(seen[i].left-seen[i-half].left) >= 7 {
				darts++
				i += 2 * half
			}
		}
		if darts == 0 {
			t.Errorf("seed %d: the Fish never darted in three minutes", seed)
		}
	}
}

func abs(n int) int { return max(n, -n) }

func TestTailSwishesFasterWhenSwimmingFaster(t *testing.T) {
	const window = 10 // frames, half a second
	for seed := range uint64(5) {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(80, 24)
		seen := h.watch(3 * time.Minute)

		var still, fast []int // tail flips per window
		for i := window; i < len(seen); i += window {
			flips := 0
			for j := i - window + 1; j <= i; j++ {
				if seen[j].tail != seen[j-1].tail && seen[j].facingLeft == seen[j-1].facingLeft {
					flips++
				}
			}
			switch moved := abs(seen[i].left - seen[i-window].left); {
			case moved == 0:
				still = append(still, flips)
			case moved >= 3:
				fast = append(fast, flips)
			}
		}
		if len(still) == 0 || len(fast) == 0 {
			t.Fatalf("seed %d: saw %d still and %d fast half-seconds", seed, len(still), len(fast))
		}
		if s, f := mean(still), mean(fast); f < 2*s {
			t.Errorf("seed %d: the tail flips %.1f times per half-second swimming fast, %.1f when still", seed, f, s)
		}
	}
}

func mean(ns []int) float64 {
	sum := 0
	for _, n := range ns {
		sum += n
	}
	return float64(sum) / float64(len(ns))
}

func TestResizingKeepsTheFishInsideTheTank(t *testing.T) {
	for seed := range uint64(5) {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(160, 50)
		sizes := []struct{ w, h int }{{30, 10}, {120, 40}, {40, 12}, {160, 50}, {31, 11}, {80, 24}, {30, 10}}
		for round := range 20 {
			// Let the Fish wander somewhere, often near an edge, then shrink
			// or grow the window under it.
			h.watch(time.Duration(1+round%4) * 2 * time.Second)
			size := sizes[round%len(sizes)]
			h.resize(size.w, size.h)
			if s, ok := findFish(h.view()); !ok || !s.intact || s.top+3 > floorRow(h.view()) {
				t.Fatalf("seed %d: after resizing to %dx%d the Fish is outside the Tank:\n%s",
					seed, size.w, size.h, strings.Join(h.view(), "\n"))
			}
		}
	}
}

func TestTheSameSeedSwimsTheSameWay(t *testing.T) {
	run := func(seed uint64) []string {
		h := newHarness(t, withSeed(seed), named("Bubbles"))
		h.resize(80, 24)
		var frames []string
		for range 400 {
			h.advance(app.FrameInterval)
			frames = append(frames, strings.Join(h.view(), "\n"))
		}
		return frames
	}
	a, b, c := run(7), run(7), run(8)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("frame %d differs between two runs with the same seed:\n%s\n\n%s", i, a[i], b[i])
		}
	}
	if strings.Join(a, "") == strings.Join(c, "") {
		t.Error("two different seeds swam exactly the same way")
	}
}
