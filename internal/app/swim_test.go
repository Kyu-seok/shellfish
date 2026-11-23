package app_test

import (
	"testing"
	"time"
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
