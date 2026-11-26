package tank

import (
	"math"

	"github.com/Kyu-seok/shellfish/internal/fish"
)

// The user playing with the Fish: Pats, and its Curiosity about the Hand.
// Neither changes its Hunger or Mood.

// hand is the Hand: the user's pointer while it is over the Tank.
type hand struct {
	col, row int
	still    float64 // seconds since it last moved
}

// Pat is the user touching the Fish: it wiggles and, unless it is Starving,
// sends up a heart, no sooner than PatHeartGap after the last one a Pat sent
// up. A Fish on its way out can't be patted.
func (t *Tank) Pat() {
	f := &t.fish
	if f.leaving != nil {
		return
	}
	f.wiggle = WiggleTime
	if d := &t.decor; t.mood != fish.Starving && d.patHeartIn <= 0 {
		d.patHeartIn = PatHeartGap
		t.flourish(f.x+float64(t.body.Width/2), t.fishRow()-1, Heart)
	}
}

// wiggle advances a Pat's wiggle by s seconds: the tail swishes and the
// body bobs faster while it lasts.
func (t *Tank) wiggle(s float64) {
	f := &t.fish
	t.decor.patHeartIn = max(t.decor.patHeartIn-s, 0)
	if f.wiggle <= 0 {
		return
	}
	f.wiggle = max(f.wiggle-s, 0)
	f.tail += s * WiggleTailRate
	f.bob += s * 2 * math.Pi * WiggleBobRate
}

// MoveHand puts the Hand at the cell col, row. It is the Hand moving only
// if that is somewhere new.
func (t *Tank) MoveHand(col, row int) {
	if t.hand != nil && t.hand.col == col && t.hand.row == row {
		return
	}
	t.hand = &hand{col: col, row: row}
}

// LoseHand takes the Hand away, as when the pointer leaves the Tank.
func (t *Tank) LoseHand() { t.hand = nil }

// curious reports whether the Fish has Curiosity now: the Hand is over the
// Tank and has moved within CuriosityFade, and the Fish is staying, not
// going for Food and not Starving.
func (t *Tank) curious() bool {
	f := t.fish
	return t.hand != nil && t.hand.still < CuriosityFade &&
		f.leaving == nil && !f.chasing && t.mood != fish.Starving
}

// visitHand starts a cruise to somewhere near the Hand, as quick as the Fish's
// Mood makes its cruises, in place of a cruise to anywhere. It reports false, and starts nothing, if it finds
// nowhere near it can swim to.
func (t *Tank) visitHand() bool {
	f := &t.fish
	e := energies[t.mood]
	maxX := float64(t.width - t.body.Width)
	highest, lowest := t.swimRange(e.low)
	highest = max(highest, highestFish)
	lowest -= BobAmplitude
	for range TargetTries {
		tx := float64(t.hand.col) - float64(t.body.Width)/2 + between(t.rand, -CuriousCols, CuriousCols)
		ty := float64(t.hand.row) - float64(t.body.Height)/2 + between(t.rand, -CuriousRows, CuriousRows)
		tx, ty = clamp(tx, 0, maxX), clamp(ty, highest, lowest)
		if t.pathClear(f.x, f.y, tx, ty) {
			t.setOff(tx, ty, CruiseSpeed*e.speed, Accel*e.speed)
			return true
		}
	}
	return false
}
