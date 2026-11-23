package tank

import (
	"math"
	"slices"
)

// Stalk is one clump of seaweed growing up from the floor: a stalk of kelp
// in the KelpWidth columns from KelpCol, and a short tuft of grass right
// beside it in the TuftWidth columns from TuftCol. It sways within those
// columns.
type Stalk struct {
	KelpCol, TuftCol int
	// Height is how many rows the kelp stands, and Tuft how many the grass.
	Height, Tuft int
	// Sway is where the stalk is in its sway, in turns from 0 up to 1. It
	// moves on continuously.
	Sway float64
}

// stalk is a Stalk as the simulation keeps it.
type stalk struct {
	col          int     // the first of the SeaweedWidth columns it stands in
	height, tuft int     // the kelp's height and the grass's, in rows
	tuftLeft     bool    // the grass is left of the kelp, not right
	phase        float64 // where in its sway it starts, in turns
}

// kelpCol and tuftCol are the first columns the kelp and the grass stand in.
func (st stalk) kelpCol() int {
	if st.tuftLeft {
		return st.col + TuftWidth
	}
	return st.col
}

func (st stalk) tuftCol() int {
	if st.tuftLeft {
		return st.col
	}
	return st.col + KelpWidth
}

// heightIn is how high the stalk stands in the columns from c0 to c1: the
// kelp's height over its columns, the tuft's over the grass's, and 0 if the
// stalk stands in none of them.
func (st stalk) heightIn(c0, c1 int) int {
	over := func(col, width int) bool { return c0 < col+width && c1 >= col }
	h := 0
	if over(st.kelpCol(), KelpWidth) {
		h = st.height
	}
	if over(st.tuftCol(), TuftWidth) {
		h = max(h, st.tuft)
	}
	return h
}

// growSeaweed plants stalks along the floor, never so tall that the Fish
// can't swim over them, and with at least SeaweedGap columns of floor between
// them so the Fish can reach Food resting anywhere.
func (t *Tank) growSeaweed() {
	t.seaweed = t.seaweed[:0]
	tallest := min(SeaweedMaxHeight, t.waterRows()-t.body.Height-FishClearance)
	if tallest < 1 || t.width < SeaweedGap+SeaweedWidth {
		return
	}
	shortest := min(SeaweedMinHeight, tallest)
	// gaps are the empty columns before each stalk, and after the last one.
	var gaps []int
	if n := (t.width - SeaweedGap) / (SeaweedGap + SeaweedWidth); n > 0 {
		gaps = make([]int, n+1)
		for i := range gaps {
			gaps[i] = SeaweedGap
		}
	} else {
		// No room for gaps on both sides: one stalk stands against a wall.
		gaps = []int{0, SeaweedGap}
		if t.rand.IntN(2) == 0 {
			gaps = []int{SeaweedGap, 0}
		}
	}
	spare := t.width - SeaweedWidth*(len(gaps)-1)
	for _, gap := range gaps {
		spare -= gap
	}
	for ; spare > 0; spare-- {
		i := t.rand.IntN(len(gaps))
		if gaps[i] == 0 { // a stalk against the wall stays there
			i = (i + 1) % len(gaps)
		}
		gaps[i]++
	}
	col := 0
	for _, gap := range gaps[:len(gaps)-1] {
		col += gap
		t.seaweed = append(t.seaweed, stalk{
			col:    col,
			height: shortest + t.rand.IntN(tallest-shortest+1),
			tuft:   min(GrassHeight, tallest),
			// The grass faces the middle of the Tank, so the Fish has room
			// to reach Food resting on it, even by a wall.
			tuftLeft: 2*col+SeaweedWidth > t.width,
			phase:    t.rand.Float64(),
		})
		col += SeaweedWidth
	}
}

// tallestStalk is the height of the tallest stalk's kelp or grass, or 0
// with no seaweed.
func (t *Tank) tallestStalk() int {
	tallest := 0
	for _, st := range t.seaweed {
		tallest = max(tallest, st.height, st.tuft)
	}
	return tallest
}

// placeShells lays the shells on the sand for the Tank's layout, one at
// each of shellSpots, nudged to the nearest column clear of the seaweed.
// They stay put until the next resize.
func (t *Tank) placeShells() {
	t.shells = t.shells[:0]
	for _, spot := range shellSpots {
		want := t.width*spot.num/spot.den + spot.nudge
		for d := 0; d < 2*t.width; d++ {
			x := want + (d+1)/2*(1-2*(d%2)) // want, want+1, want-1, want+2, ...
			if t.shellFits(x) {
				t.shells = append(t.shells, x)
				break
			}
		}
	}
}

// shellFits reports whether a shell can rest at column x: inside the Tank,
// on a column of its own, and ShellClearance clear of the SeaweedWidth
// columns every stalk stands and sways in.
func (t *Tank) shellFits(x int) bool {
	if x < 0 || x >= t.width {
		return false
	}
	for _, s := range t.shells {
		if s == x {
			return false
		}
	}
	for _, st := range t.seaweed {
		if x >= st.col-ShellClearance && x <= st.col+SeaweedWidth-1+ShellClearance {
			return false
		}
	}
	return true
}

// snapshotScenery adds the seaweed and shells to snap.
func (t *Tank) snapshotScenery(snap *Snapshot) {
	for _, st := range t.seaweed {
		snap.Seaweed = append(snap.Seaweed, Stalk{
			KelpCol: st.kelpCol(),
			TuftCol: st.tuftCol(),
			Height:  st.height,
			Tuft:    st.tuft,
			Sway:    math.Mod(t.clock/SeaweedSwayPeriod+st.phase, 1),
		})
	}
	snap.Shells = slices.Clone(t.shells)
}
