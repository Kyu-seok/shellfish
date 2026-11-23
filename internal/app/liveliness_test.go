package app_test

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

const (
	heart = "♥"
	note  = "♪"
)

// bubbleSizes are a bubble's glyphs, smallest first.
var bubbleSizes = []string{"°", "o", "O"}

func isBubble(c string) bool { return c == "°" || c == "o" || c == "O" }

// inMood launches on a Fish left alone long enough to be in mood: Content
// (just named), Hungry (12 hours away) or Starving (26 hours away).
func inMood(t *testing.T, mood string, opts ...option) *harness {
	t.Helper()
	h := newHarness(t, append(opts, named("Bubbles"))...)
	switch mood {
	case "Hungry":
		h = h.reopen(12 * time.Hour)
	case "Starving":
		h = h.reopen(26 * time.Hour)
	}
	return h
}

// droopSettle is how long a Starving Fish may take to droop to the lower
// part of the Tank from anywhere in it.
const droopSettle = 4 * time.Second

// checkLow fails the test if the Fish rises out of the lower 40% of the rows
// its top can reach in any of seen. Its bob may lift it one row more.
func checkLow(t *testing.T, h *harness, seen []sighting, about string) {
	t.Helper()
	lowestTop := floorRow(h.view()) - body.Height // a Fish resting just above the sand
	highest := int(math.Ceil(0.6*float64(lowestTop))) - 1
	for i, s := range seen {
		if s.top < highest {
			t.Fatalf("%s, frame %d: a Starving Fish swam up to row %d, above the lower 40%% of the water (row %d):\n%s",
				about, i, s.top, highest, h.screen())
		}
	}
}

func TestAStarvingFishStaysLowInTheTank(t *testing.T) {
	sizes := []struct{ w, h int }{{80, 24}, {40, 14}, {60, 18}}
	for seed := range uint64(6) {
		size := sizes[seed%uint64(len(sizes))]
		h := inMood(t, "Starving", withSeed(seed))
		h.resize(size.w, size.h)
		// Time to droop from the middle of the Tank, and for the welcome-back
		// greeting, which can hide a Fish high in a short Tank, to go.
		h.advance(droopSettle + time.Second)

		checkLow(t, h, h.watch(45*time.Second), fmt.Sprintf("seed %d, %dx%d", seed, size.w, size.h))
	}
}

func TestAFishDroopsAsSoonAsItIsStarving(t *testing.T) {
	for seed := range uint64(4) {
		h := inMood(t, "Hungry", withSeed(seed)) // Hunger 50
		h.resize(80, 24)
		h.watch(10 * time.Second)                    // let it wander anywhere
		h.clock.now = h.clock.now.Add(8 * time.Hour) // Hunger 83: Starving
		h.watch(droopSettle)

		checkLow(t, h, h.watch(10*time.Second), fmt.Sprintf("seed %d", seed))
	}
}

// travel is how many columns the Fish swam, frame to frame, in all. Rows
// are left out: the Fish bobs up and down even while it hovers.
func travel(seen []sighting) int {
	n := 0
	for i := 1; i < len(seen); i++ {
		n += abs(seen[i].left - seen[i-1].left)
	}
	return n
}

func TestMoodSetsHowLivelyTheFishIs(t *testing.T) {
	moods := []string{"Content", "Hungry", "Starving"}
	moved := map[string]int{}
	for _, mood := range moods {
		for seed := range uint64(3) {
			h := inMood(t, mood, withSeed(seed))
			h.resize(80, 24)
			moved[mood] += travel(h.watch(time.Minute))
		}
	}
	for i := 1; i < len(moods); i++ {
		livelier, slower := moods[i-1], moods[i]
		if 2*moved[livelier] < 3*moved[slower] {
			t.Errorf("a %s Fish moved %d cells and a %s one %d: want the %s one well ahead",
				livelier, moved[livelier], slower, moved[slower], livelier)
		}
	}
}

func TestTheFishBlinksEveryFewSeconds(t *testing.T) {
	second := int(time.Second / app.FrameInterval)
	for _, mood := range []string{"Content", "Hungry", "Starving"} {
		for seed := range uint64(2) {
			h := inMood(t, mood, withSeed(seed))
			h.resize(80, 24)
			seen := h.watch(20 * time.Second)

			var blinks []int // frames each blink started
			for i, s := range seen {
				if s.blinking && (i == 0 || !seen[i-1].blinking) {
					blinks = append(blinks, i)
				}
			}
			if len(blinks) == 0 || blinks[0] >= 6*second {
				t.Fatalf("%s, seed %d: no blink in the first 6 seconds (blinks at frames %v)", mood, seed, blinks)
			}
			for i := 1; i < len(blinks); i++ {
				if gap := blinks[i] - blinks[i-1]; gap < 3*second || gap > 6*second+1 {
					t.Errorf("%s, seed %d: blinks %d frames apart, want 3–6 seconds", mood, seed, gap)
				}
			}
		}
	}
}

// faintHeart matches a heart drawn faint, as it fades away.
var faintHeart = regexp.MustCompile(`\x1b\[([0-9]+;)*2(;[0-9]+)*m[^\x1b]*` + heart)

// heartRow is the row of the first heart in the water, or -1.
func heartRow(rows []string) int {
	for y, row := range rows[:floorRow(rows)] {
		if strings.Contains(row, heart) {
			return y
		}
	}
	return -1
}

func TestAHeartFloatsUpAfterTheFishEats(t *testing.T) {
	h := hungerAfter(t, 12*time.Hour) // Hungry, and still Hungry after a pellet
	fish, ok := findFish(h.rawView())
	if !ok {
		t.Fatalf("no Fish in the Tank:\n%s", h.screen())
	}
	// Mid-water, across the Tank from the Fish, so there is room above.
	col := 5
	if fish.left < 40 {
		col = 74
	}
	h.click(col, 10)

	_, frames := h.meal(30 * time.Second)
	for i, frame := range frames[:len(frames)-1] {
		if strings.Contains(frame, heart) {
			t.Fatalf("a heart showed at frame %d, before the Fish ate:\n%s", i, frame)
		}
	}
	first := -1
	for range 5 {
		if first = heartRow(h.view()); first >= 0 {
			break
		}
		h.advance(app.FrameInterval)
	}
	if first < 0 {
		t.Fatalf("no heart after the Fish ate:\n%s", h.screen())
	}

	highest, faded := first, false
	for range int(3 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		if y := heartRow(h.view()); y >= 0 {
			highest = min(highest, y)
		}
		for _, row := range h.rawView() {
			faded = faded || faintHeart.MatchString(row)
		}
	}
	if highest > first-2 {
		t.Errorf("the heart started on row %d and rose no higher than row %d", first, highest)
	}
	if !faded {
		t.Error("the heart never faded")
	}
	if heartRow(h.view()) >= 0 {
		t.Errorf("the heart is still there 3 seconds after the meal:\n%s", h.screen())
	}
}

func TestAContentFishShowsHeartsAndNotesNowAndThen(t *testing.T) {
	for seed := range uint64(3) {
		h := inMood(t, "Content", withSeed(seed))
		h.resize(80, 24)
		shown := false
		for _, rows := range h.frames(30 * time.Second) {
			screen := strings.Join(rows, "\n")
			shown = shown || strings.Contains(screen, heart) || strings.Contains(screen, note)
		}
		if !shown {
			t.Errorf("seed %d: a Content Fish showed no heart or note in 30 seconds", seed)
		}
	}
}

// newBubbles counts the bubbles in cur that appeared out of nowhere in the
// water, rather than rising from the cell below them or from behind
// something: bubbles blown by the Fish. Bubbles rising from the floor first
// show on the lowest row of water, which is left out. The Fish's box hides
// what is behind it even where its drawing shows a space.
func newBubbles(prev, cur []string) (n int, glyphs []string) {
	floor := floorRow(cur)
	minX, minY, maxX, maxY, seen := fishBounds(prev)
	hidden := func(x, y int) bool {
		return seen && x >= minX && x <= maxX && y >= minY && y <= maxY
	}
	for y := 0; y < floor-1; y++ {
		was, below, now := cells(prev[y]), cells(prev[y+1]), cells(cur[y])
		for x, c := range now {
			if !isBubble(c) {
				continue
			}
			if x < len(was) && was[x] != " " || x < len(below) && below[x] != " " || hidden(x, y) || hidden(x, y+1) {
				continue // it rose from there, or was hidden behind something
			}
			n++
			glyphs = append(glyphs, c)
		}
	}
	return n, glyphs
}

func TestAContentFishBlowsBubblesThatGrowAsTheyRise(t *testing.T) {
	for seed := range uint64(3) {
		h := inMood(t, "Content", withSeed(seed))
		h.resize(80, 24)
		frames := h.frames(30 * time.Second)

		blown, large := 0, false
		for i := 1; i < len(frames); i++ {
			n, glyphs := newBubbles(frames[i-1], frames[i])
			blown += n
			for _, g := range glyphs {
				if g != bubbleSizes[0] {
					t.Errorf("seed %d frame %d: a new bubble is %q, want the smallest, %q", seed, i, g, bubbleSizes[0])
				}
			}
			large = large || strings.Contains(strings.Join(frames[i], ""), bubbleSizes[2])
		}
		if blown == 0 {
			t.Errorf("seed %d: a Content Fish blew no bubbles in 30 seconds", seed)
		}
		if !large {
			t.Errorf("seed %d: no bubble grew to %q", seed, bubbleSizes[2])
		}
	}
}

func TestAStarvingFishBlowsNoBubblesButBubblesStillRise(t *testing.T) {
	for seed := range uint64(3) {
		h := inMood(t, "Starving", withSeed(seed))
		h.resize(80, 24)
		frames := h.frames(30 * time.Second)

		nearTop, nearFloor := false, false
		for i, rows := range frames {
			if i > 0 {
				if n, _ := newBubbles(frames[i-1], rows); n > 0 {
					t.Fatalf("seed %d frame %d: a Starving Fish blew a bubble:\n%s", seed, i, strings.Join(rows, "\n"))
				}
			}
			for y, row := range rows[:floorRow(rows)] {
				if strings.ContainsAny(row, "°oO") {
					nearTop = nearTop || y < 4
					nearFloor = nearFloor || y > floorRow(rows)-4
				}
			}
			screen := strings.Join(rows, "\n")
			if strings.Contains(screen, heart) || strings.Contains(screen, note) {
				t.Fatalf("seed %d frame %d: a Starving Fish showed a heart or note:\n%s", seed, i, screen)
			}
		}
		if !nearTop || !nearFloor {
			t.Errorf("seed %d: bubbles were seen near the floor: %v, near the top: %v; want them rising all the way",
				seed, nearFloor, nearTop)
		}
	}
}
