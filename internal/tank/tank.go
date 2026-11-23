// Package tank simulates the Tank: where the Fish is, how it moves, and the
// seaweed it swims around. It advances in time steps and is deterministic
// given its random source.
package tank

import (
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/Kyu-seok/shellfish/internal/fish"
)

// Snapshot is what the Tank looks like at one moment, for drawing.
type Snapshot struct {
	// Width and Height are the Tank's size in cells. The last row is the floor.
	Width, Height int
	Fish          Fish
	Seaweed       []Stalk
	Food          []Food
	Bubbles       []Bubble
	// Flourishes are the hearts and notes floating up from the Fish.
	Flourishes []Flourish
	// Clock is how many seconds the Tank has run, for Scenery that drifts.
	Clock float64
}

// Fish is where the Fish is drawn.
type Fish struct {
	// Col and Row are the top-left cell of its FishWidth×FishHeight box.
	Col, Row   int
	FacingLeft bool
	// TailUp alternates as the tail swishes.
	TailUp bool
	// MouthOpen is set just before the Fish eats the Food it is going for.
	MouthOpen bool
	// Blinking is set while its eyes are shut.
	Blinking bool
}

// Food is one piece of Food in the Tank.
type Food struct {
	Col, Row int
	Kind     fish.FoodKind
	// Fading is set for Food that has rested on the floor nearly long enough
	// to be gone.
	Fading bool
}

// Meal is a piece of Food the Fish ate, at the cell where its mouth met it.
// It is the moment for anything that celebrates eating to start from.
type Meal struct {
	Kind     fish.FoodKind
	Col, Row int
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
	fish          swimmer
	seaweed       []stalk
	food          []food
	mood          fish.Mood // how the Fish feels, which sets its energy
	decor         decor
}

type food struct {
	x, y   float64 // x is its column; y sinks smoothly
	kind   fish.FoodKind
	landed bool
	rested float64 // seconds spent resting on the floor
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

// swimmer is the Fish as the simulation moves it.
type swimmer struct {
	x, y       float64 // top-left of its box
	vx, vy     float64
	facingLeft bool
	mode       mode
	timer      float64 // seconds left hovering or holding, or spent on this trip
	tx, ty     float64 // wander target
	top, accel float64 // top speed and acceleration for this trip
	tail       float64 // tail swish phase
	bob        float64 // bob phase, in radians
	// chasing is set while the Fish is going for Food. Its target is then
	// where its mouth meets that Food.
	chasing   bool
	mouthOpen bool
}

// New fills a width×height Tank, the floor being its last row, with seaweed
// and a Fish in the middle.
func New(r *rand.Rand, width, height int) *Tank {
	t := &Tank{rand: r}
	t.Resize(width, height)
	t.fish.x = float64((t.width - FishWidth + 1) / 2) // centres the 11-cell drawing
	t.fish.y = float64((t.waterRows() - FishHeight) / 2)
	t.keepFishInside()
	// Decorations draw from their own source, seeded last so the seaweed is
	// the same as without them.
	t.decor = newDecor(rand.New(rand.NewPCG(r.Uint64(), r.Uint64())))
	return t
}

// Resize changes the Tank's size, regrows the seaweed and moves the Fish back
// inside if it no longer fits.
func (t *Tank) Resize(width, height int) {
	t.width, t.height = width, height
	t.growSeaweed()
	t.keepFishInside()
	t.trimDecor()
	t.fish.mode, t.fish.timer, t.fish.chasing = hovering, 0, false
	t.food = slices.DeleteFunc(t.food, func(fd food) bool { return int(fd.x) >= t.width })
	for i := range t.food {
		fd := &t.food[i]
		// Food that no longer has the floor or seaweed it rested on sinks again.
		rest := float64(t.restRow(int(fd.x)))
		fd.y = min(fd.y, rest)
		fd.landed = fd.landed && fd.y == rest
	}
}

// Drop puts Food of kind k into the Tank at the cell col, row. Food dropped
// into the floor or seaweed rests on top of it. It reports false, and drops
// nothing, if the cell is outside the Tank or the Tank already holds MaxFood.
func (t *Tank) Drop(k fish.FoodKind, col, row int) bool {
	if len(t.food) >= MaxFood || col < 0 || col >= t.width || row < 0 || row >= t.height {
		return false
	}
	rest := t.restRow(col)
	y := float64(min(max(row, SurfaceRows), rest))
	t.food = append(t.food, food{x: float64(col), y: y, kind: k, landed: row >= rest})
	return true
}

// DropNearTop drops Food of kind k at a random column on the top row of
// water, just below the surface, as Drop does.
func (t *Tank) DropNearTop(k fish.FoodKind) bool {
	if t.width <= 0 {
		return false
	}
	return t.Drop(k, t.rand.IntN(t.width), 0)
}

// restRow is the row Food comes to rest on in column col: just above the
// floor, or on top of a stalk of seaweed.
func (t *Tank) restRow(col int) int {
	row := t.waterRows() - 1
	for _, st := range t.seaweed {
		if col == st.col || col == st.col+1 {
			row = t.waterRows() - 1 - st.height
		}
	}
	return max(row, SurfaceRows)
}

// Step advances the simulation by dt for a Fish in mood, which sets how
// lively it is. A Fish that isn't Full goes for the nearest Food it can
// reach and eats it; a Full one ignores Food. Step returns what the Fish
// ate, at most one piece, so the caller can tell whether it is Full before
// the next step.
func (t *Tank) Step(dt time.Duration, mood fish.Mood, full bool) []Meal {
	s := dt.Seconds()
	t.clock += s
	t.mood = mood
	t.decorate(s, mood)
	t.settleFood(s)
	f := &t.fish
	f.bob += s * 2 * math.Pi / BobPeriod
	t.chase(full)
	t.droop()
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
		if f.chasing {
			break // it stays on the Food until it eats it
		}
		ahead := f.tx - f.x
		if f.facingLeft {
			ahead = -ahead
		}
		if ahead < ArriveDistance && math.Abs(f.ty-f.y) < ArriveDistance || f.timer > MaxTrip {
			f.mode, f.timer = hovering, t.hoverTime()
		}
	}
	t.move(s)
	f.tail += s * (TailBaseRate + TailRatePerSpeed*math.Hypot(f.vx, f.vy))
	if full {
		return nil
	}
	meals := t.eat()
	for _, m := range meals {
		t.ate(m)
	}
	return meals
}

// settleFood sinks Food towards where it comes to rest, and takes away Food
// that has rested for FoodRestTime.
func (t *Tank) settleFood(s float64) {
	for i := range t.food {
		fd := &t.food[i]
		if fd.landed {
			fd.rested += s
			continue
		}
		rest := float64(t.restRow(int(fd.x)))
		fd.y = min(fd.y+fd.kind.SinkSpeed()*s, rest)
		fd.landed = fd.y == rest
	}
	t.food = slices.DeleteFunc(t.food, func(fd food) bool { return fd.rested >= FoodRestTime })
}

// chase points a Fish that isn't Full at the nearest Food it can reach,
// aiming for where the Food will be by the time the Fish gets there. A Fish
// facing the wrong way slows, holds and turns first, as it does when
// wandering.
func (t *Tank) chase(full bool) {
	f := &t.fish
	f.mouthOpen = false
	tx, ty, left, ok := t.nearestFood(full)
	if !ok {
		if f.chasing {
			f.chasing = false
			f.mode, f.timer = hovering, t.hoverTime()/2
		}
		return
	}
	f.chasing = true
	f.tx, f.ty, f.top, f.accel = tx, ty, ChaseSpeed, ChaseAccel
	if !t.pathClear(f.x, f.y, tx, ty) {
		// Rise over the seaweed first, then come down to the Food.
		f.ty = min(f.ty, t.aboveSeaweed())
	}
	f.mouthOpen = math.Abs(tx-f.x) <= MouthOpenReach && math.Abs(ty-f.y) <= MouthOpenReach && left == f.facingLeft
	switch {
	case f.mode == turning:
	case t.travelFacing(tx, left) != f.facingLeft:
		f.mode, f.timer = turning, TurnHold
	default:
		f.mode = swimming
	}
}

// travelFacing is which way the Fish should face on its way to x, where it
// will face left or not to eat: towards x, until it is close enough to back
// up the rest of the way facing the Food.
func (t *Tank) travelFacing(x float64, left bool) bool {
	if math.Abs(x-t.fish.x) <= BackUpDistance {
		return left
	}
	return x < t.fish.x
}

// turns is how many times the Fish must turn around to eat at x facing left.
func (t *Tank) turns(x float64, left bool) int {
	n, facing := 0, t.fish.facingLeft
	if travel := t.travelFacing(x, left); travel != facing {
		n, facing = n+1, travel
	}
	if left != facing {
		n++
	}
	return n
}

// nearestFood is where the Fish should be to eat the nearest Food it can
// reach, counting the time it takes to turn around, and which way it should
// face there.
func (t *Tank) nearestFood(full bool) (x, y float64, left, ok bool) {
	if full {
		return 0, 0, false, false
	}
	f := t.fish
	best := math.Inf(1)
	for _, fd := range t.food {
		for _, l := range []bool{f.facingLeft, !f.facingLeft} {
			tx, ty, reachable := t.intercept(fd, l)
			if !reachable {
				continue
			}
			cost := math.Hypot(tx-f.x, (ty-f.y)/VerticalScale) + TurnCost*float64(t.turns(tx, l))
			if cost < best {
				best, x, y, left, ok = cost, tx, ty, l, true
			}
		}
	}
	return x, y, left, ok
}

// intercept is where the Fish should be, facing left or not, to meet fd as
// it sinks.
func (t *Tank) intercept(fd food, left bool) (x, y float64, ok bool) {
	f := t.fish
	fy := fd.y
	for range InterceptSteps {
		x, y, ok = t.mouthOn(fd.x, fy, left)
		if !ok || fd.landed {
			return x, y, ok
		}
		eta := max(math.Abs(x-f.x)/ChaseSpeed, math.Abs(y-f.y)/(ChaseSpeed*VerticalScale))
		fy = min(fd.y+fd.kind.SinkSpeed()*eta, float64(t.restRow(int(fd.x))))
	}
	return x, y, ok
}

// mouthOn finds a place for the Fish, facing left or not, where its mouth
// reaches the cell at col, row.
func (t *Tank) mouthOn(col, row float64, left bool) (x, y float64, ok bool) {
	// Right on the Food, or a cell short of it either way, out of the way of
	// a wall or a stalk.
	for _, dx := range []float64{0, -1, 1} {
		x := clamp(col-mouthCol(left)+dx, 0, float64(t.width-FishWidth))
		y := clamp(row-1, highestFish, float64(t.waterRows()-FishHeight)-BobAmplitude)
		// Rise until clear of the seaweed.
		for !t.fits(x, y) && y > highestFish {
			y = max(y-RiseStep, highestFish)
		}
		if t.fits(x, y) && reaches(x, y, left, col, row) {
			return x, y, true
		}
	}
	return 0, 0, false
}

// mouthCol is the column of the Fish's mouth in its box.
func mouthCol(left bool) float64 {
	if left {
		return 0
	}
	return FishWidth - 2
}

// reaches reports whether a Fish at x, y facing left or right can eat Food at
// col, row.
func reaches(x, y float64, left bool, col, row float64) bool {
	return math.Abs(x+mouthCol(left)-col) <= EatReachCols && math.Abs(y+1-row) <= EatReachRows
}

// eat has the Fish eat the first piece of Food within reach of its mouth, if
// any, and close its mouth on it.
func (t *Tank) eat() []Meal {
	f := &t.fish
	for i, fd := range t.food {
		if reaches(f.x, f.y, f.facingLeft, fd.x, fd.y) {
			f.mouthOpen = false
			t.food = slices.Delete(t.food, i, i+1)
			return []Meal{{Kind: fd.kind, Col: int(fd.x), Row: int(math.Round(fd.y))}}
		}
	}
	return nil
}

// aboveSeaweed is the lowest the Fish can be and still swim over every stalk.
func (t *Tank) aboveSeaweed() float64 {
	tallest := 0
	for _, st := range t.seaweed {
		tallest = max(tallest, st.height)
	}
	return max(float64(t.waterRows()-tallest-FishHeight)-BobAmplitude, highestFish)
}

// Snapshot returns what the Tank looks like now.
func (t *Tank) Snapshot() Snapshot {
	f := t.fish
	snap := Snapshot{
		Width:  t.width,
		Height: t.height,
		Clock:  t.clock,
		Fish: Fish{
			Col:        int(math.Round(f.x)),
			Row:        int(t.fishRow()),
			FacingLeft: f.facingLeft,
			TailUp:     int(f.tail)%2 == 1,
			MouthOpen:  f.mouthOpen,
		},
	}
	for _, fd := range t.food {
		snap.Food = append(snap.Food, Food{
			Col:    int(fd.x),
			Row:    int(math.Round(fd.y)),
			Kind:   fd.kind,
			Fading: fd.rested >= FoodRestTime-FoodFadeTime,
		})
	}
	for _, st := range t.seaweed {
		snap.Seaweed = append(snap.Seaweed, Stalk{
			Col:    st.col,
			Height: st.height,
			Sway:   int(t.clock*2/SeaweedSwayPeriod+st.phase)%2 == 1,
		})
	}
	t.snapshotDecor(&snap)
	return snap
}

// startTrip picks somewhere to swim to: now and then a quick dart ahead,
// otherwise a cruise to anywhere in the Tank, or anywhere low in it for a
// Starving Fish. How fast it goes depends on its Mood.
func (t *Tank) startTrip() {
	f := &t.fish
	e := energies[t.mood]
	if t.rand.Float64() < e.dartChance {
		dir := 1.0
		if f.facingLeft {
			dir = -1
		}
		tx := f.x + dir*between(t.rand, DartMin, DartMax)
		ty := f.y + between(t.rand, -DartRise, DartRise)
		if t.pathClear(f.x, f.y, tx, ty) {
			t.setOff(tx, ty, DartSpeed*e.speed, DartAccel*e.speed)
			return
		}
	}
	highest, lowest := t.swimRange(e.low)
	for range TargetTries {
		tx := t.rand.Float64() * float64(t.width-FishWidth)
		ty := between(t.rand, highest, lowest)
		if t.pathClear(f.x, f.y, tx, ty) {
			t.setOff(tx, ty, CruiseSpeed*e.speed, Accel*e.speed)
			return
		}
	}
	f.timer = 1
}

// swimRange is the highest and lowest the Fish's top wanders between: the
// whole water, or only the bottom LowWater of it when low.
func (t *Tank) swimRange(low bool) (highest, lowest float64) {
	lowest = float64(t.waterRows() - FishHeight)
	if low {
		highest = lowest * (1 - LowWater)
	}
	return highest, lowest
}

// droop sends a Starving Fish that is above the lower part of the Tank, and
// not going for Food or already on its way down, straight down there at
// DroopSpeed, without pausing first.
func (t *Tank) droop() {
	f := &t.fish
	e := energies[t.mood]
	highest, lowest := t.swimRange(e.low)
	if !e.low || f.chasing || f.mode == turning || f.y >= highest ||
		f.mode == swimming && f.ty >= highest {
		return
	}
	for ty := lowest; ty >= highest; ty-- {
		if t.pathClear(f.x, f.y, f.x, ty) {
			t.setOff(f.x, ty, DroopSpeed, Accel)
			return
		}
	}
}

// hoverTime is how long the Fish pauses between trips: longer the worse it
// feels.
func (t *Tank) hoverTime() float64 {
	return between(t.rand, HoverMin, HoverMax) * energies[t.mood].pause
}

// setOff starts a trip to tx, ty, turning around first if it is behind.
func (t *Tank) setOff(tx, ty, top, accel float64) {
	f := &t.fish
	f.tx, f.ty, f.top, f.accel = tx, ty, top, accel
	f.mode, f.timer = swimming, 0
	if behind := (tx < f.x) != f.facingLeft; behind && math.Abs(tx-f.x) >= ArriveDistance {
		f.mode, f.timer = turning, TurnHold
	}
}

// steer accelerates the Fish towards its target: it eases in, coasts at its
// top speed and eases out as it arrives. It only swims forwards, except to
// back up a little to line its mouth up with Food.
func (t *Tank) steer(s float64) {
	f := &t.fish
	want := arriveSpeed(f.tx-f.x, f.top, f.accel)
	back := 0.0
	if f.chasing {
		back = ReverseSpeed
	}
	if f.facingLeft {
		want = min(want, back)
	} else {
		want = max(want, -back)
	}
	f.vx = approach(f.vx, want, f.accel*s)
	f.vy = approach(f.vy, arriveSpeed(f.ty-f.y, f.top*VerticalScale, f.accel*VerticalScale), f.accel*VerticalScale*s)
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

// fits reports whether the Fish at x, y stays inside the water and clear of
// the seaweed wherever its bob takes it.
func (t *Tank) fits(x, y float64) bool {
	return t.boxFits(x, y-BobAmplitude) && t.boxFits(x, y+BobAmplitude)
}

// boxFits reports whether the Fish's box at x, y is inside the water and
// clear of the seaweed.
func (t *Tank) boxFits(x, y float64) bool {
	col, top := int(math.Round(x)), int(math.Round(y))
	bottom := top + FishHeight - 1
	if col < 0 || col+FishWidth > t.width || top < SurfaceRows || bottom >= t.waterRows() {
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
	f.y = clamp(f.y, highestFish, float64(t.waterRows()-FishHeight)-BobAmplitude)
	for !t.fits(f.x, f.y) && f.y > highestFish {
		f.y = max(f.y-1, highestFish)
	}
	f.vx, f.vy = 0, 0
}

// growSeaweed plants stalks along the floor, never so tall that the Fish
// can't swim over them, and with at least SeaweedGap columns of floor between
// them so the Fish can reach Food resting anywhere.
func (t *Tank) growSeaweed() {
	t.seaweed = t.seaweed[:0]
	tallest := min(SeaweedMaxHeight, t.waterRows()-FishHeight-FishClearance)
	if tallest < 1 || t.width < SeaweedGap+2 {
		return
	}
	shortest := min(SeaweedMinHeight, tallest)
	// gaps are the empty columns before each stalk, and after the last one.
	var gaps []int
	if n := (t.width - SeaweedGap) / (SeaweedGap + 2); n > 0 {
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
	spare := t.width - 2*(len(gaps)-1)
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
			phase:  t.rand.Float64() * 2,
		})
		col += 2
	}
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(v, hi)) }

// between is a random number from lo up to hi.
func between(r *rand.Rand, lo, hi float64) float64 { return lo + r.Float64()*(hi-lo) }

// highestFish is the highest the top of the Fish's box goes, so that its bob
// never takes it into the surface.
const highestFish = SurfaceRows + BobAmplitude
