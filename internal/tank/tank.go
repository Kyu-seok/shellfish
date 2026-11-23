// Package tank simulates the Tank: where the Fish is, how it moves, and the
// seaweed it swims around. It advances in time steps and is deterministic
// given its random source.
package tank

import (
	"math"
	"math/rand/v2"
	"time"
)

// Snapshot is what the Tank looks like at one moment, for drawing.
type Snapshot struct {
	// Width and Height are the Tank's size in cells. The last row is the floor.
	Width, Height int
	Fish          Fish
	Seaweed       []Stalk
}

// Fish is where the Fish is drawn.
type Fish struct {
	// Col and Row are the top-left cell of its FishWidth×FishHeight box.
	Col, Row   int
	FacingLeft bool
	// TailUp alternates as the tail swishes.
	TailUp bool
}

// Stalk is one piece of seaweed growing up from the floor. It is two cells
// wide, Col and Col+1, and sways between them.
type Stalk struct {
	Col, Height int
	// Sway flips as the stalk sways.
	Sway bool
}

// Tank is the running simulation.
type Tank struct {
	rand          *rand.Rand
	width, height int
	clock         float64 // seconds simulated
	fish          fish
	seaweed       []stalk
}

type stalk struct {
	col, height int
	phase       float64 // where in its sway it starts
}

// mode is what the Fish is doing.
type mode int

const (
	hovering mode = iota
	swimming
	// turning brakes to a stop, holds for TurnHold, then mirrors and swims.
	turning
)

type fish struct {
	x, y       float64 // top-left of its box
	vx, vy     float64
	facingLeft bool
	mode       mode
	timer      float64 // seconds left hovering or holding, or spent on this trip
	tx, ty     float64 // wander target
	tail       float64 // tail swish phase
}

// New fills a width×height Tank, the floor being its last row, with seaweed
// and a Fish in the middle.
func New(r *rand.Rand, width, height int) *Tank {
	t := &Tank{rand: r}
	t.Resize(width, height)
	t.fish.x = float64(t.width-FishWidth) / 2
	t.fish.y = float64(t.waterRows()-FishHeight) / 2
	t.keepFishInside()
	return t
}

// Resize changes the Tank's size, regrows the seaweed and moves the Fish back
// inside if it no longer fits.
func (t *Tank) Resize(width, height int) {
	t.width, t.height = width, height
	t.growSeaweed()
	t.keepFishInside()
	t.fish.mode, t.fish.timer = hovering, 0
}

// Step advances the simulation by dt.
func (t *Tank) Step(dt time.Duration) {
	s := dt.Seconds()
	t.clock += s
	f := &t.fish
	switch f.mode {
	case hovering:
		f.vx = approach(f.vx, 0, Accel*s)
		f.vy = approach(f.vy, 0, Accel*VerticalScale*s)
		f.timer -= s
		if f.timer <= 0 {
			t.startTrip()
		}
	case turning:
		f.vx = approach(f.vx, 0, Accel*s)
		f.vy = approach(f.vy, 0, Accel*VerticalScale*s)
		if f.vx == 0 {
			f.timer -= s
		}
		if f.timer <= 0 {
			f.facingLeft = !f.facingLeft
			f.mode, f.timer = swimming, 0
		}
	case swimming:
		f.timer += s
		t.steer(s)
		ahead := f.tx - f.x
		if f.facingLeft {
			ahead = -ahead
		}
		if ahead < ArriveDistance && math.Abs(f.ty-f.y) < ArriveDistance || f.timer > MaxTrip {
			f.mode, f.timer = hovering, 0
		}
	}
	t.move(s)
	f.tail += s * (TailBaseRate + TailRatePerSpeed*math.Hypot(f.vx, f.vy))
}

// Snapshot returns what the Tank looks like now.
func (t *Tank) Snapshot() Snapshot {
	f := t.fish
	snap := Snapshot{
		Width:  t.width,
		Height: t.height,
		Fish: Fish{
			Col:        int(math.Round(f.x)),
			Row:        int(math.Round(f.y)),
			FacingLeft: f.facingLeft,
			TailUp:     int(f.tail)%2 == 1,
		},
	}
	for _, st := range t.seaweed {
		snap.Seaweed = append(snap.Seaweed, Stalk{
			Col:    st.col,
			Height: st.height,
			Sway:   int(t.clock*2/SeaweedSwayPeriod+st.phase)%2 == 1,
		})
	}
	return snap
}

// startTrip picks somewhere to swim to.
func (t *Tank) startTrip() {
	f := &t.fish
	for range TargetTries {
		tx := t.rand.Float64() * float64(t.width-FishWidth)
		ty := t.rand.Float64() * float64(t.waterRows()-FishHeight)
		if t.pathClear(f.x, f.y, tx, ty) {
			f.tx, f.ty = tx, ty
			f.mode, f.timer = swimming, 0
			if behind := (tx < f.x) != f.facingLeft; behind && math.Abs(tx-f.x) >= ArriveDistance {
				f.mode, f.timer = turning, TurnHold
			}
			return
		}
	}
	f.timer = 1
}

// steer accelerates the Fish towards its target: it eases in, coasts at its
// top speed and eases out as it arrives. It only ever swims forwards.
func (t *Tank) steer(s float64) {
	f := &t.fish
	want := arriveSpeed(f.tx-f.x, CruiseSpeed, Accel)
	if f.facingLeft {
		want = min(want, 0)
	} else {
		want = max(want, 0)
	}
	f.vx = approach(f.vx, want, Accel*s)
	f.vy = approach(f.vy, arriveSpeed(f.ty-f.y, CruiseSpeed*VerticalScale, Accel*VerticalScale), Accel*VerticalScale*s)
}

// arriveSpeed is the velocity that covers d and stops there without braking
// harder than decel, capped at top.
func arriveSpeed(d, top, decel float64) float64 {
	return math.Copysign(min(top, math.Sqrt(2*decel*math.Abs(d))), d)
}

// approach moves v towards target by at most step.
func approach(v, target, step float64) float64 {
	if v < target {
		return min(v+step, target)
	}
	return max(v-step, target)
}

// move applies the Fish's velocity, stopping it short of anything solid.
func (t *Tank) move(s float64) {
	f := &t.fish
	nx, ny := f.x+f.vx*s, f.y+f.vy*s
	switch {
	case t.fits(nx, ny):
		f.x, f.y = nx, ny
	case t.fits(f.x, ny):
		f.y, f.vx = ny, 0
	case t.fits(nx, f.y):
		f.x, f.vy = nx, 0
	default:
		f.vx, f.vy = 0, 0
	}
}

// waterRows is how many rows of water there are above the floor.
func (t *Tank) waterRows() int { return t.height - 1 }

// fits reports whether the Fish's box at x, y is inside the water and clear
// of the seaweed.
func (t *Tank) fits(x, y float64) bool {
	col, top := int(math.Round(x)), int(math.Round(y))
	bottom := top + FishHeight - 1
	if col < 0 || col+FishWidth > t.width || top < 0 || bottom >= t.waterRows() {
		return false
	}
	for _, st := range t.seaweed {
		overlaps := st.col <= col+FishWidth-1 && st.col+1 >= col
		if overlaps && bottom >= t.waterRows()-st.height {
			return false
		}
	}
	return true
}

// pathClear reports whether the Fish can swim in a straight line between
// two points.
func (t *Tank) pathClear(x0, y0, x1, y1 float64) bool {
	steps := int(math.Hypot(x1-x0, y1-y0)*2) + 1
	for i := range steps + 1 {
		k := float64(i) / float64(steps)
		if !t.fits(x0+(x1-x0)*k, y0+(y1-y0)*k) {
			return false
		}
	}
	return true
}

// keepFishInside moves the Fish to the nearest place it fits: inside the
// walls, then up and over any seaweed it would be caught in.
func (t *Tank) keepFishInside() {
	f := &t.fish
	f.x = clamp(f.x, 0, float64(t.width-FishWidth))
	f.y = clamp(f.y, 0, float64(t.waterRows()-FishHeight))
	for !t.fits(f.x, f.y) && f.y > 0 {
		f.y = max(math.Round(f.y)-1, 0)
	}
	f.vx, f.vy = 0, 0
}

// growSeaweed plants stalks along the floor, never so tall that the Fish
// can't swim over them.
func (t *Tank) growSeaweed() {
	t.seaweed = t.seaweed[:0]
	tallest := min(SeaweedMaxHeight, t.waterRows()-FishHeight-FishClearance)
	if tallest < 1 || t.width < 4 {
		return
	}
	shortest := min(SeaweedMinHeight, tallest)
	n := max(1, t.width/SeaweedSpacing)
	slot := t.width / n
	for i := range n {
		// One stalk somewhere in each slot, clear of the slot's edges so
		// neighbours never touch.
		col := i*slot + 1 + t.rand.IntN(max(slot-3, 1))
		t.seaweed = append(t.seaweed, stalk{
			col:    min(col, t.width-2),
			height: shortest + t.rand.IntN(tallest-shortest+1),
			phase:  t.rand.Float64() * 2,
		})
	}
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(v, hi)) }
