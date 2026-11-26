package app_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

// farewell is what the status line shows while Bubbles swims away.
const farewell = "Bye, Bubbles!"

func TestYSaysByeThenOpensThePicker(t *testing.T) {
	for _, opts := range [][]option{{named("Bubbles")}, {named("Bubbles"), monochrome()}} {
		h := newHarness(t, opts...)
		h.resize(100, 24)
		h.press("r")
		h.press("y")

		if got := h.statusLine(); got != farewell {
			t.Errorf("after y the status line is %q, want %q", got, farewell)
		}
		h.advance(app.FarewellDuration - app.FrameInterval)
		if got := h.statusLine(); got != farewell {
			t.Errorf("just before the farewell ends the status line is %q, want %q", got, farewell)
		}
		if strings.Contains(h.screen(), "Species: ‹") {
			t.Fatalf("the picker opened before the farewell ended:\n%s", h.screen())
		}
		h.advance(app.FrameInterval)
		if !strings.Contains(h.screen(), "Species: ‹ Clownfish ›") {
			t.Errorf("the picker didn't open on the first Species after the farewell:\n%s", h.screen())
		}
	}
}

func TestTheFishSwimsOutPastTheNearerSideDuringTheFarewell(t *testing.T) {
	for seed := range uint64(4) {
		for _, opts := range [][]option{{}, {monochrome()}} {
			h := newHarness(t, append(opts, named("Bubbles"), withSeed(seed))...)
			h.resize(100, 24)
			h.advance(time.Duration(seed) * 7 * time.Second) // somewhere along its swim
			left, _, right, _, ok := fishBounds(h.view())
			if !ok {
				t.Fatalf("seed %d: no Fish to release:\n%s", seed, h.screen())
			}
			toLeft := left < 100-1-right
			h.press("r")
			h.press("y")

			last, lastSeen := -1, -1 // the nearer edge's side of the Fish, as last seen
			frames := h.frames(app.FarewellDuration - app.FrameInterval)
			for i, rows := range frames {
				l, _, r, _, ok := fishBounds(rows)
				if !ok {
					continue
				}
				side := l
				if !toLeft {
					side = 99 - r
				}
				if last >= 0 && side > last {
					t.Errorf("seed %d frame %d: the Fish turned back from the nearer edge:\n%s", seed, i, strings.Join(rows, "\n"))
				}
				last, lastSeen = side, i
			}
			if lastSeen < 0 || lastSeen == len(frames)-1 {
				t.Errorf("seed %d: the Fish is still in the Tank at the end of the farewell:\n%s", seed, strings.Join(frames[len(frames)-1], "\n"))
			} else if last != 0 {
				t.Errorf("seed %d: the Fish was last seen %d cells from the nearer edge, want it to swim out past it", seed, last)
			}
			if got := h.statusLine(); got != farewell {
				t.Errorf("seed %d: the farewell ended early: %q", seed, got)
			}
		}
	}
}

func TestKeysAndClicksDoNothingDuringTheFarewell(t *testing.T) {
	for _, key := range []string{"q", "esc", "f", " ", "2", "r", "y", "n", "right", "enter", "click"} {
		t.Run(key, func(t *testing.T) {
			h := newHarness(t, named("Bubbles"))
			h.resize(100, 24)
			h.press("r")
			h.press("y")
			if key == "click" {
				h.click(7, 3)
			} else {
				h.press(key)
			}

			if h.quitRequested() {
				t.Fatalf("%q during the farewell quit the app", key)
			}
			if got := h.statusLine(); got != farewell {
				t.Errorf("after %q the status line is %q, want %q", key, got, farewell)
			}
			h.advance(app.FrameInterval)
			if n := h.count(pellet) + h.count(flake) + h.count(worm); n != 0 {
				t.Errorf("%q during the farewell dropped Food:\n%s", key, h.screen())
			}
			h.advance(app.FarewellDuration)
			if !strings.Contains(h.screen(), "Species: ‹ Clownfish ›") {
				t.Errorf("after %q the farewell didn't end on the picker's first Species:\n%s", key, h.screen())
			}
		})
	}
}

func TestCtrlCDuringTheFarewellQuitsKeepingTheOldFish(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(100, 24)
	h.press("r")
	h.press("y")
	h.advance(app.FarewellDuration / 2)
	h.press("ctrl+c")

	if !h.quitRequested() {
		t.Fatal("Ctrl-C during the farewell didn't quit")
	}
	again := newHarness(t, withDataDir(h.dataDir), at(h.clock.now.Add(time.Minute)))
	again.resize(100, 24)
	again.advance(app.FrameInterval)
	if !strings.HasPrefix(again.statusLine(), "Bubbles ·") {
		t.Errorf("after quitting during the farewell, the next launch shows %q, want Bubbles' Tank", again.statusLine())
	}
	if _, ok := findFish(again.rawView()); !ok {
		t.Errorf("Bubbles didn't come back to the Tank:\n%s", again.screen())
	}
}
