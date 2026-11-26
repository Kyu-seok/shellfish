package app_test

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Kyu-seok/shellfish/internal/app"
)

func TestAClickOnTheFishPatsItInsteadOfDroppingFood(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	h.patFish()
	h.advance(200 * time.Millisecond) // before a Content Fish shows a heart by itself
	if got := h.count(heart); got != 1 {
		t.Errorf("a Pat sent up %d hearts, want 1:\n%s", got, h.screen())
	}
	for range int(10 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		if got := h.count(pellet); got != 0 {
			t.Fatalf("a click on the Fish dropped Food:\n%s", h.screen())
		}
	}
}

// A click on the Fish is a Pat whatever else is in the Tank, even Food.
func TestAClickOnTheFishIsAPatEvenWithFoodInTheWater(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles")) // Full, so the Food stays put
	h.resize(80, 24)
	h.click(2, h.bottom()) // on the sand in a corner, away from the Fish
	if got := h.count(pellet); got != 1 {
		t.Fatalf("the Tank holds %d pellets, want 1:\n%s", got, h.screen())
	}

	h.patFish()
	h.advance(200 * time.Millisecond)
	if got := h.count(pellet); got != 1 {
		t.Errorf("after a click on the Fish the Tank holds %d pellets, want still 1:\n%s", got, h.screen())
	}
	if got := h.count(heart); got != 1 {
		t.Errorf("a Pat with Food in the water sent up %d hearts, want 1:\n%s", got, h.screen())
	}
}

func TestAClickRightBesideTheFishDropsFood(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles")) // Full, so it leaves the Food alone
	h.resize(80, 24)
	minX, minY, maxX, maxY, ok := fishBounds(h.view())
	if !ok {
		t.Fatalf("no Fish:\n%s", h.screen())
	}
	// A cell of water inside the box around the Fish, between its fins.
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if h.cellAt(x, y) != " " {
				continue
			}
			h.click(x, y)
			if got := h.cellAt(x, y); got != pellet {
				t.Errorf("a click on the water at (%d, %d), beside the Fish, drew %q, want a pellet:\n%s", x, y, got, h.screen())
			}
			return
		}
	}
	t.Fatalf("no water inside the box around the Fish:\n%s", h.screen())
}

// Patting fast sends up a heart a second at most, so it can't flood the Tank.
func TestPattingFastSendsUpAtMostAHeartASecond(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)

	for range 10 {
		h.patFish()
		h.advance(app.FrameInterval)
	}
	if got := h.count(heart); got != 1 {
		t.Errorf("ten Pats in half a second sent up %d hearts, want 1:\n%s", got, h.screen())
	}
	h.advance(time.Second)
	h.patFish()
	h.advance(app.FrameInterval)
	if got := h.count(heart); got != 2 {
		t.Errorf("a Pat a second later left %d hearts, want 2:\n%s", got, h.screen())
	}
}

// A Starving Fish wiggles its tail when patted, but sends up no heart: it
// wants feeding first.
func TestAPattedStarvingFishWigglesWithoutAHeart(t *testing.T) {
	t.Parallel()
	patted, alone := twins(t, 22*time.Hour)
	if got := patted.fishMood(); got != "Starving" {
		t.Fatalf("the Fish is %s, want Starving", got)
	}
	patted.patFish()
	// How many frames each Fish's tail moved in.
	swishes := map[*harness]int{}
	last := map[*harness]float64{}
	for range int(600 * time.Millisecond / app.FrameInterval) {
		for _, h := range []*harness{patted, alone} {
			h.advance(app.FrameInterval)
			if s, ok := h.find(h.rawView()); ok && s.intact {
				if s.tailLift != last[h] {
					swishes[h]++
				}
				last[h] = s.tailLift
			}
		}
		if got := patted.count(heart); got != 0 {
			t.Fatalf("a Pat on a Starving Fish sent up a heart:\n%s", patted.screen())
		}
	}
	if swishes[patted] <= swishes[alone] {
		t.Errorf("a patted Starving Fish's tail moved in %d frames, one left alone's in %d: want it to wiggle more",
			swishes[patted], swishes[alone])
	}
}

func TestTheFishWandersNearAMovingHand(t *testing.T) {
	t.Parallel()
	for _, after := range []time.Duration{6 * time.Hour, 12 * time.Hour} { // Content and Hungry
		h := hungerAfter(t, after)
		h.resize(120, 30)

		middles := h.waveHand(105, 8, 30*time.Second)
		for _, m := range middles[len(middles)-int(8*time.Second/app.FrameInterval):] {
			if abs(m.col-105) > 16 || abs(m.row-8) > 5 {
				t.Fatalf("after %v, with the Hand moving at (105, 8) for 30s, the middle of the Fish was at %v:\n%s",
					after, m, h.screen())
			}
		}
	}
}

// The Fish loses interest in a Hand that stays still, and comes back when it
// moves again.
func TestCuriosityFadesWhileTheHandIsStill(t *testing.T) {
	t.Parallel()
	h := hungerAfter(t, 6*time.Hour)
	h.resize(120, 30)
	h.waveHand(105, 8, 20*time.Second)

	strayed := false
	for range int(60 * time.Second / app.FrameInterval) { // the Hand stays at (105, 8)
		h.advance(app.FrameInterval)
		if s, ok := h.find(h.rawView()); ok && h.middle(s).col < 105-40 {
			strayed = true
			break
		}
	}
	if !strayed {
		t.Fatalf("with the Hand still for a minute, the Fish never wandered away from it:\n%s", h.screen())
	}

	middles := h.waveHand(105, 8, 30*time.Second)
	if m := middles[len(middles)-1]; abs(m.col-105) > 16 {
		t.Errorf("30s after the Hand moved again the middle of the Fish was at %v, far from it:\n%s", m, h.screen())
	}
}

func TestFoodComesBeforeTheHand(t *testing.T) {
	t.Parallel()
	h := hungerAfter(t, 12*time.Hour)
	h.resize(120, 30)
	h.waveHand(105, 8, 15*time.Second)

	h.click(4, 4)
	ate := false
	for i := 0; i < int(20*time.Second/app.FrameInterval) && !ate; i++ {
		h.hover(105+i%2, 8) // the Hand keeps moving
		_, ate = h.eats(app.FrameInterval)
	}
	if !ate {
		t.Errorf("with the Hand moving across the Tank, the Fish never ate the Food:\n%s", h.screen())
	}
}

// Where the Hand draws no Curiosity, a Tank with it moving runs exactly as
// one without it.
func TestTheHandIsIgnored(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		after time.Duration
		setUp func(h *harness)
		row   func(h *harness) int
	}{
		{"by a Starving Fish", 22 * time.Hour, func(*harness) {}, func(*harness) int { return 8 }},
		{"over the status line", 6 * time.Hour, func(*harness) {}, func(h *harness) int { return len(h.view()) - 1 }},
		{"while releasing is asked about", 6 * time.Hour, func(h *harness) { h.press("r") }, func(*harness) int { return 8 }},
	} {
		moved, alone := twins(t, tc.after)
		for _, h := range []*harness{moved, alone} {
			h.resize(120, 30)
			tc.setUp(h)
		}
		for i := range int(20 * time.Second / app.FrameInterval) {
			moved.hover(100+i%7, tc.row(moved))
			moved.advance(app.FrameInterval)
			alone.advance(app.FrameInterval)
			if moved.screen() != alone.screen() {
				t.Fatalf("%s, the Hand changed the Tank at frame %d:\n%s\nwant:\n%s", tc.name, i, moved.screen(), alone.screen())
			}
		}
	}
}

// The Hand is gone as soon as releasing is asked about, even if the pointer
// was moving over the Tank just before and stays there.
func TestAskingToReleaseTakesTheHandAway(t *testing.T) {
	t.Parallel()
	stayed, gone := twins(t, 6*time.Hour)
	for _, h := range []*harness{stayed, gone} {
		h.resize(120, 30)
		h.hover(15, 20) // far from where the Fish starts
		h.advance(app.FrameInterval)
	}
	gone.hover(15, len(gone.view())-1) // off the Tank, onto the status line
	for _, h := range []*harness{stayed, gone} {
		h.press("r")
	}
	for i := range int(20 * time.Second / app.FrameInterval) {
		stayed.advance(app.FrameInterval)
		gone.advance(app.FrameInterval)
		if stayed.screen() != gone.screen() {
			t.Fatalf("while releasing was asked about, a Hand left over the Tank changed it at frame %d:\n%s\nwant:\n%s",
				i, stayed.screen(), gone.screen())
		}
	}
}

// Only the pointer moving with no button held is the Hand. In a terminal that
// reports nothing else, the Fish wanders just as it did before there was one.
func TestOnlyMovingThePointerIsTheHand(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		do   func(h *harness)
	}{
		{"a click", func(h *harness) { h.click(100, 8) }},
		{"a drag", func(h *harness) {
			h.send(tea.MouseMsg{X: 100, Y: 8, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
		}},
	} {
		// Full, so the Fish leaves the Food a click drops alone.
		done, gone := newHarness(t, named("Bubbles"), withSeed(7)), newHarness(t, named("Bubbles"), withSeed(7))
		for _, h := range []*harness{done, gone} {
			h.resize(120, 30)
			tc.do(h)
		}
		gone.hover(100, len(gone.view())-1) // any Hand it made is gone again
		for i := range int(10 * time.Second / app.FrameInterval) {
			done.advance(app.FrameInterval)
			gone.advance(app.FrameInterval)
			if done.screen() != gone.screen() {
				t.Fatalf("%s made the Fish curious about it at frame %d:\n%s\nwant:\n%s", tc.name, i, done.screen(), gone.screen())
			}
		}
	}
}
