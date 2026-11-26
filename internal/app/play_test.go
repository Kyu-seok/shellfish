package app_test

import (
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

// patFish clicks on the Fish's eye, which is always drawn, and fails the test
// if there is no Fish to click on.
func (h *harness) patFish() {
	h.t.Helper()
	s, ok := h.find(h.rawView())
	if !ok {
		h.t.Fatalf("no Fish to pat:\n%s", h.screen())
	}
	x0, y0, _, _ := eyeArea(h.species.Art(), h.body(), s.facingLeft)
	h.click(s.left+x0, s.top+y0)
}

// twins are two Tanks that start alike, with the same seed and Hunger, so
// whatever one does that the other doesn't is the only thing that can make
// them differ.
func twins(t *testing.T, after time.Duration) (a, b *harness) {
	t.Helper()
	return hungerAfter(t, after, withSeed(7)), hungerAfter(t, after, withSeed(7))
}

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

// A Starving Fish wiggles when patted, but sends up no heart: it wants
// feeding first.
func TestAPattedStarvingFishWigglesWithoutAHeart(t *testing.T) {
	t.Parallel()
	patted, left := twins(t, 22*time.Hour)
	if got := patted.fishMood(); got != "Starving" {
		t.Fatalf("the Fish is %s, want Starving", got)
	}
	patted.patFish()
	wiggled := false
	for range int(time.Second / app.FrameInterval) {
		patted.advance(app.FrameInterval)
		left.advance(app.FrameInterval)
		wiggled = wiggled || patted.screen() != left.screen()
		if got := patted.count(heart); got != 0 {
			t.Fatalf("a Pat on a Starving Fish sent up a heart:\n%s", patted.screen())
		}
	}
	if !wiggled {
		t.Errorf("a patted Starving Fish moved just as one left alone did")
	}
}

// hand keeps the Hand moving around col, row for d, a frame at a time, and
// returns where the middle of the Fish was in each frame.
func (h *harness) hand(col, row int, d time.Duration) (middles [][2]int) {
	h.t.Helper()
	for i := range int(d / app.FrameInterval) {
		h.hover(col+i/int(time.Second/app.FrameInterval)%2, row) // a column across and back each second
		h.advance(app.FrameInterval)
		if s, ok := h.find(h.rawView()); ok {
			middles = append(middles, [2]int{s.left + h.body().Width/2, s.top + h.body().Height/2})
		}
	}
	return middles
}

func TestTheFishWandersNearAMovingHand(t *testing.T) {
	t.Parallel()
	for _, after := range []time.Duration{6 * time.Hour, 12 * time.Hour} { // Content and Hungry
		h := hungerAfter(t, after)
		h.resize(120, 30)

		middles := h.hand(105, 8, 30*time.Second)
		for _, m := range middles[len(middles)-int(8*time.Second/app.FrameInterval):] {
			if abs(m[0]-105) > 16 || abs(m[1]-8) > 5 {
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
	h.hand(105, 8, 20*time.Second)

	strayed := false
	for range int(60 * time.Second / app.FrameInterval) { // the Hand stays at (105, 8)
		h.advance(app.FrameInterval)
		if s, ok := h.find(h.rawView()); ok && s.left+h.body().Width/2 < 105-40 {
			strayed = true
			break
		}
	}
	if !strayed {
		t.Fatalf("with the Hand still for a minute, the Fish never wandered away from it:\n%s", h.screen())
	}

	middles := h.hand(105, 8, 30*time.Second)
	if m := middles[len(middles)-1]; abs(m[0]-105) > 16 {
		t.Errorf("30s after the Hand moved again the middle of the Fish was at %v, far from it:\n%s", m, h.screen())
	}
}

func TestFoodComesBeforeTheHand(t *testing.T) {
	t.Parallel()
	h := hungerAfter(t, 12*time.Hour)
	h.resize(120, 30)
	h.hand(105, 8, 15*time.Second)

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

// Where the Hand has no Curiosity to draw, a Tank with it moving runs
// exactly as one without it.
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
		moved, left := twins(t, tc.after)
		for _, h := range []*harness{moved, left} {
			h.resize(120, 30)
			tc.setUp(h)
		}
		for i := range int(20 * time.Second / app.FrameInterval) {
			moved.hover(100+i%7, tc.row(moved))
			moved.advance(app.FrameInterval)
			left.advance(app.FrameInterval)
			if moved.screen() != left.screen() {
				t.Fatalf("%s, the Hand changed the Tank at frame %d:\n%s\nwant:\n%s", tc.name, i, moved.screen(), left.screen())
			}
		}
	}
}
