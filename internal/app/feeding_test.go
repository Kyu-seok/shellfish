package app_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

// Food glyphs, as the Tank draws each Food Kind.
const (
	pellet = "•"
	flake  = "*"
	worm   = "ʃ"
)

func TestClickDropsFoodAtThatExactCell(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	h.click(7, 3)
	if got := h.cellAt(7, 3); got != pellet {
		t.Errorf("cell (7, 3) after a click = %q, want a pellet %q:\n%s", got, pellet, h.screen())
	}
}

func TestFOrSpaceDropsFoodAtTheTopOfTheTank(t *testing.T) {
	for _, key := range []string{"f", " "} {
		h := newHarness(t, named("Bubbles"))
		h.resize(80, 24)

		h.press(key)
		if got := strings.Count(h.view()[waterTop], pellet); got != 1 {
			t.Errorf("after pressing %q the top row of water holds %d pellets, want 1:\n%s", key, got, h.screen())
		}
		if got := h.count(pellet); got != 1 {
			t.Errorf("after pressing %q the Tank holds %d pellets, want 1:\n%s", key, got, h.screen())
		}
	}
}

func TestFDropsFoodInDifferentPlaces(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	columns := map[int]bool{}
	for range 6 {
		h.press("f")
		columns[strings.Index(h.view()[waterTop], pellet)] = true
		h.advance(time.Second) // let it sink out of the way
	}
	if len(columns) < 3 {
		t.Errorf("six presses of f dropped Food in only %d columns", len(columns))
	}
}

func TestNumberKeysSelectTheFoodKind(t *testing.T) {
	for _, tc := range []struct {
		key, glyph, name string
	}{
		{"1", pellet, "pellet"},
		{"2", flake, "flake"},
		{"3", worm, "worm"},
	} {
		h := newHarness(t, named("Bubbles"))
		h.resize(100, 24)
		h.press("3") // start from something else, to see the key change it
		h.press("2")
		h.press(tc.key)

		if want := "Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱ · Age 0 days · " + tc.glyph + " " + tc.name; !strings.HasPrefix(h.statusLine(), want) {
			t.Errorf("after pressing %s the status line is %q, want it to start %q", tc.key, h.statusLine(), want)
		}
		h.click(40, 2)
		if got := h.cellAt(40, 2); got != tc.glyph {
			t.Errorf("after pressing %s a click drops %q, want a %s %q", tc.key, got, tc.name, tc.glyph)
		}
	}
}

// A terminal hands over keys typed in quick succession as one message, so
// tapping or holding a key must still act once per key.
func TestKeysTypedQuicklyEachTakeEffect(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(100, 24)

	h.press("3f")
	if !strings.Contains(h.statusLine(), "· "+worm+" worm") {
		t.Errorf("after typing 3f quickly the status line is %q, want the worm selected", h.statusLine())
	}
	if got := h.count(worm); got != 1 {
		t.Errorf("after typing 3f quickly the Tank holds %d worms, want 1:\n%s", got, h.screen())
	}

	h.press("ff")
	if got := h.count(worm); got != 3 {
		t.Errorf("after typing ff quickly the Tank holds %d worms, want 3:\n%s", got, h.screen())
	}
}

func TestPelletIsSelectedAtFirst(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(100, 24)

	if !strings.Contains(h.statusLine(), "· "+pellet+" pellet") {
		t.Errorf("status line is %q, want the pellet selected", h.statusLine())
	}
}

// In an 80-column window there is room for the Food Kind's name or the key
// hint, not both: the Food Kind is shown by its glyph and the hint stays.
func TestSelectedFoodKindLeavesRoomForTheKeyHint(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)
	h.press("3")

	status := h.statusLine()
	if want := "Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱ · Age 0 days · " + worm + " "; !strings.HasPrefix(status, want) {
		t.Errorf("status line is %q, want it to start %q", status, want)
	}
	if !strings.HasSuffix(status, keyHint) {
		t.Errorf("status line is %q, want it to end with the key hint", status)
	}
}

// hungerAfter launches on a Fish left alone for d, so it is that hungry,
// and waits for the first periodic save, so the save file shows it. Hunger
// rises 100 a day: 12 hours away is Hunger 50.
func hungerAfter(t *testing.T, d time.Duration, opts ...option) *harness {
	t.Helper()
	h := newHarness(t, append(opts, named("Bubbles"))...).reopen(d)
	h.resize(80, 24)
	h.advance(app.SaveInterval)
	return h
}

func TestEachFoodKindLowersHungerByItsOwnAmount(t *testing.T) {
	for _, tc := range []struct {
		key  string
		name string
		want float64
	}{
		{"1", "pellet", 40},
		{"2", "flake", 25},
		{"3", "worm", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := hungerAfter(t, 12*time.Hour) // Hunger 50
			h.press(tc.key)
			h.press("f")

			// Hunger also rises a little while the Fish waits and swims.
			if got, _ := h.meal(30 * time.Second); got < tc.want || got > tc.want+0.2 {
				t.Errorf("after eating a %s Hunger is %.3f, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestEatingNeverTakesHungerBelowFull(t *testing.T) {
	h := hungerAfter(t, 2*time.Hour+24*time.Minute) // Hunger 10
	h.press("3")
	h.press("f")

	if got, _ := h.meal(30 * time.Second); got != 0 {
		t.Errorf("after a worm on Hunger 10, Hunger is %v, want 0 (Full)", got)
	}
	if !strings.Contains(h.statusLine(), "Hunger ▱▱▱▱▱▱▱▱▱▱") {
		t.Errorf("status line is %q, want an empty Hunger meter", h.statusLine())
	}
}

func TestTheFishOpensItsMouthJustBeforeItEats(t *testing.T) {
	h := hungerAfter(t, 12*time.Hour) // Hungry
	fish, ok := findFish(h.rawView())
	if !ok {
		t.Fatalf("no Fish in the Tank:\n%s", h.screen())
	}
	// Drop the Food at the top, across the Tank from the Fish.
	col := 5
	if fish.left < 40 {
		col = 74
	}
	h.click(col, 0)

	// Frame by frame until the saved Hunger drops: until the Fish eats.
	before := readSave(t, h.dataDir).Hunger
	var open []bool // whether its mouth was open, each frame
	for len(open) < int(30*time.Second/app.FrameInterval) && readSave(t, h.dataDir).Hunger >= before {
		h.advance(app.FrameInterval)
		s, _ := findFish(h.rawView())
		open = append(open, s.eating)
	}
	if readSave(t, h.dataDir).Hunger >= before {
		t.Fatalf("the Fish ate nothing in 30 seconds:\n%s", h.screen())
	}
	last := slices.Index(open, true)
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
}

func TestMoodAndEyeUpdateTheMomentTheFishEats(t *testing.T) {
	h := hungerAfter(t, 12*time.Hour) // Hungry
	h.press("3")
	h.press("f")

	h.meal(30 * time.Second)
	if h.fishMood() != "Content" {
		t.Errorf("right after a worm the Fish does not look Content:\n%s", h.screen())
	}
	if !strings.Contains(h.statusLine(), "Hunger ▱▱▱▱▱▱▱▱▱▱") {
		t.Errorf("right after a worm the status line is %q", h.statusLine())
	}
}

func TestTheFishIsSavedAfterEachMeal(t *testing.T) {
	h := hungerAfter(t, 12*time.Hour)
	for _, want := range []float64{40, 30} {
		h.press("f")
		// meal reads the save file within a frame of eating, well before the
		// next periodic save.
		if got, _ := h.meal(20 * time.Second); got < want || got > want+0.2 {
			t.Errorf("saved Hunger after a pellet is %.3f, want %v", got, want)
		}
	}
}

// bottom is the lowest row of water, where Food comes to rest on the sand.
func TestAFullFishIgnoresFood(t *testing.T) {
	h := newHarness(t, named("Bubbles")) // born Full
	h.resize(80, 24)
	h.click(5, 2)

	// Pellets sink about a row a second: by now it has long come to rest.
	h.advance(25 * time.Second)
	if got := h.cellAt(5, h.bottom()); got != pellet {
		t.Errorf("a Full Fish's pellet is not resting on the sand below where it was dropped:\n%s", h.screen())
	}
	if got := readSave(t, h.dataDir).Hunger; got != 0 {
		t.Errorf("saved Hunger = %v, want 0: the Fish ate", got)
	}
}

func TestAFishEatsAgainOnceItIsNoLongerFull(t *testing.T) {
	h := newHarness(t, named("Bubbles")) // born Full
	h.resize(80, 24)
	h.click(5, 2)
	h.advance(5 * time.Second)

	h.clock.now = h.clock.now.Add(12 * time.Hour) // as if the Tank had stayed open half a day
	h.advance(app.FrameInterval)                  // which the periodic save notes: Hunger 50
	if got, _ := h.meal(30 * time.Second); got < 40 || got > 40.1 {
		t.Errorf("after eating the pellet it ignored while Full, Hunger is %.3f, want 40", got)
	}
}

func TestTheTankHoldsAtMostEightPiecesOfFood(t *testing.T) {
	h := newHarness(t, named("Bubbles")) // Full, so it eats none of them
	h.resize(80, 24)

	for i := range 9 {
		h.click(4+4*i, 2)
	}
	if got := h.count(pellet); got != 8 {
		t.Errorf("after 9 clicks the Tank holds %d pellets, want 8:\n%s", got, h.screen())
	}
	if got := h.cellAt(36, 2); got != " " {
		t.Errorf("the 9th click dropped %q, want it ignored", got)
	}
	h.press("f")
	if got := h.count(pellet); got != 8 {
		t.Errorf("f dropped a 9th pellet:\n%s", h.screen())
	}
}

func TestFoodOnTheFloorFadesAfterThirtySeconds(t *testing.T) {
	h := newHarness(t, named("Bubbles")) // Full, so it leaves the Food alone
	h.resize(80, 24)
	h.click(5, h.bottom()) // dropped on the sand, so it rests at once

	seen, faint, colour := false, false, ""
	for range int(30 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		// The Fish may swim in front of it now and then.
		here, st := h.cellAt(5, h.bottom()) == pellet, h.styleAt(5, h.bottom())
		if here && !st.faint {
			colour = st.fg
		}
		if here && st.faint && st.fg != colour {
			t.Fatalf("the fading pellet is drawn in %q, want its colour %q kept", st.fg, colour)
		}
		seen = seen || here
		faint = faint || here && st.faint
		if h.clock.now.Sub(start) == 20*time.Second && (!seen || faint) {
			t.Fatalf("after 20 seconds on the floor the pellet is not there, or already fading:\n%s", h.screen())
		}
	}
	if !faint {
		t.Error("the pellet never faded before it went")
	}
	h.advance(app.FrameInterval)
	for range int(2 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		if h.count(pellet) != 0 {
			t.Fatalf("the pellet is still there after 30 seconds on the floor:\n%s", h.screen())
		}
	}
}

func TestTheGreetingDoesNotBlockFeeding(t *testing.T) {
	h := newHarness(t, named("Bubbles")).reopen(3 * time.Hour)
	h.resize(60, 14)
	if !strings.Contains(h.screen(), "missed you") {
		t.Fatalf("no greeting to feed through:\n%s", h.screen())
	}

	h.click(2, 9)
	h.press("f")
	if got := h.cellAt(2, 9); got != pellet {
		t.Errorf("a click while greeting dropped %q at (2, 9), want a pellet:\n%s", got, h.screen())
	}
	if got := strings.Count(h.view()[waterTop], pellet); got != 1 {
		t.Errorf("f while greeting put %d pellets on the top row of water, want 1:\n%s", got, h.screen())
	}
	if !strings.Contains(h.screen(), "missed you") {
		t.Errorf("feeding dismissed the greeting:\n%s", h.screen())
	}
}

func TestAHungryFishEatsFoodDroppedAnywhereSoon(t *testing.T) {
	const limit = 20 * time.Second
	sizes := []struct{ w, h int }{{80, 24}, {40, 12}, {120, 40}, {30, 10}, {57, 17}}
	for seed := range uint64(40) {
		size := sizes[seed%uint64(len(sizes))]
		h := hungerAfter(t, 12*time.Hour, withSeed(seed))
		h.resize(size.w, size.h)
		where := rand.New(rand.NewPCG(seed, 99))
		col, row := where.IntN(size.w), where.IntN(floorRow(h.view())+1) // any cell of the Tank, sand included
		weed := seaweedCells(h.view())
		h.click(col, row)

		_, frames := h.meal(limit)
		for i, frame := range frames {
			if seaweedCells(strings.Split(frame, "\n")) != weed {
				t.Fatalf("seed %d: going for Food at (%d, %d), the Fish swam into seaweed at frame %d:\n%s",
					seed, col, row, i, frame)
			}
		}
	}
}

func TestAClickOnTheFishDropsFoodToo(t *testing.T) {
	h := newHarness(t, named("Bubbles")) // Full, so the Food sinks past it
	h.resize(80, 24)
	fish, ok := findFish(h.rawView())
	if !ok {
		t.Fatalf("no Fish to click on:\n%s", h.screen())
	}
	x0, y0, _, _ := eyeArea(art, body, fish.facingLeft)
	eyeCol, eyeRow := fish.left+x0, fish.top+y0
	h.click(eyeCol, eyeRow)

	for range int(15 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		for row := eyeRow + 1; row <= h.bottom(); row++ {
			if h.cellAt(eyeCol, row) == pellet {
				return // it sank out from behind the Fish
			}
		}
	}
	t.Errorf("clicking on the Fish at (%d, %d) dropped no Food:\n%s", eyeCol, eyeRow, h.screen())
}

// Each Food Kind is drawn in its own colour, so they can be told apart:
// pellets brown, flakes orange, worms pink.
func TestEachFoodKindIsDrawnInItsOwnColour(t *testing.T) {
	seen := map[string]string{}
	for _, tc := range []struct {
		key, glyph, name string
		looks            func(r, g, b int) bool
	}{
		{"1", pellet, "brown pellet", func(r, g, b int) bool { return r > g && g > b && r < 0xd0 }},
		{"2", flake, "orange flake", func(r, g, b int) bool { return r >= 0xd0 && g > b && r-g >= 0x30 && g-b >= 0x30 }},
		{"3", worm, "pink worm", func(r, g, b int) bool { return r >= 0xd0 && b > g }},
	} {
		h := newHarness(t, named("Bubbles"))
		h.resize(100, 24)
		h.press(tc.key)
		h.click(40, 2)
		if got := h.cellAt(40, 2); got != tc.glyph {
			t.Fatalf("a click dropped %q, want a %s %q", got, tc.name, tc.glyph)
		}
		fg := h.styleAt(40, 2).fg
		var r, g, b int
		if _, err := fmt.Sscanf(fg, "#%02x%02x%02x", &r, &g, &b); err != nil || !tc.looks(r, g, b) {
			t.Errorf("the %s is drawn in %q", tc.name, fg)
		}
		if other, ok := seen[fg]; ok {
			t.Errorf("the %s is drawn in the same colour as the %s, %q", tc.name, other, fg)
		}
		seen[fg] = tc.name
	}
}
