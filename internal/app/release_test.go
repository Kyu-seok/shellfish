package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
	"github.com/Kyu-seok/shellfish/internal/fish"
)

// confirmation is what the status line asks when the user presses r.
const confirmation = "Release Bubbles to the sea? y/N"

func TestRAsksToReleaseTheFishByName(t *testing.T) {
	for _, opts := range [][]option{{named("Bubbles")}, {named("Bubbles"), monochrome()}} {
		h := newHarness(t, opts...)
		h.resize(80, 24)
		h.press("r")

		if got := h.statusLine(); got != confirmation {
			t.Errorf("after r the status line is %q, want %q", got, confirmation)
		}
		if h.quitRequested() {
			t.Error("r quit the app")
		}
		if strings.Contains(h.screen(), "Species: ‹") {
			t.Errorf("r opened the picker before y:\n%s", h.screen())
		}
	}
}

func TestEveryKeyButYOnlyCancelsTheConfirmation(t *testing.T) {
	for _, key := range []string{"q", "esc", "f", " ", "1", "2", "3", "r", "n", "enter"} {
		t.Run(key, func(t *testing.T) {
			h := newHarness(t, named("Bubbles"))
			h.resize(100, 24)
			h.press("r")
			h.press(key)

			if h.quitRequested() {
				t.Fatalf("%q on the confirmation quit the app", key)
			}
			if got := h.statusLine(); !strings.HasPrefix(got, "Bubbles · Hunger") || !strings.Contains(got, pellet+" pellet") {
				t.Errorf("after %q the status line is %q, want the Fish's status with the pellet still selected", key, got)
			}
			h.advance(app.FrameInterval)
			if n := h.count(pellet) + h.count(flake) + h.count(worm); n != 0 {
				t.Errorf("%q on the confirmation dropped Food:\n%s", key, h.screen())
			}
		})
	}
}

func TestAClickOnlyCancelsTheConfirmation(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(100, 24)
	h.press("r")
	h.click(7, 3)

	if got := h.cellAt(7, 3); got == pellet {
		t.Errorf("a click on the confirmation dropped Food:\n%s", h.screen())
	}
	if got := h.statusLine(); strings.Contains(got, "Release") {
		t.Errorf("a click left the confirmation up: %q", got)
	}
}

func TestCtrlCQuitsFromTheConfirmationKeepingTheFish(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(100, 24)
	h.press("r")
	h.press("ctrl+c")

	if !h.quitRequested() {
		t.Fatal("Ctrl-C on the confirmation didn't quit")
	}
	if s := readSave(t, h.dataDir); s.Name != "Bubbles" {
		t.Errorf("after quitting from the confirmation the saved Fish is %q", s.Name)
	}
}

func TestTheTankStaysLiveDuringTheConfirmation(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(100, 24)
	h.press("f")
	h.press("r")

	before := h.screen()
	h.advance(time.Minute)
	if h.screen() == before {
		t.Error("the Tank stood still for a minute under the confirmation")
	}
	if got := h.statusLine(); got != confirmation {
		t.Errorf("the confirmation went away by itself: %q", got)
	}
	if s := readSave(t, h.dataDir); !s.LastSeen.Equal(start.Add(time.Minute)) || s.Hunger <= 0 {
		t.Errorf("after a minute under the confirmation the save is from %v with Hunger %v, want an autosave at %v", s.LastSeen, s.Hunger, start.Add(time.Minute))
	}
}

// release answers y to the confirmation, so the picker opens for a new Fish.
func (h *harness) release() {
	h.t.Helper()
	h.press("r")
	h.press("y")
}

// bubbles counts the bubbles drawn in the water, leaving out the Fish.
func bubbles(rows []string) int {
	left, top, right, bottom, seen := fishBounds(rows)
	n := 0
	for y := 1; y < floorRow(rows)-1; y++ {
		for x, c := range cells(rows[y]) {
			inFish := seen && x >= left && x <= right && y >= top && y <= bottom
			if !inFish && strings.Contains("°oO", c) && c != "" {
				n++
			}
		}
	}
	return n
}

func TestYOpensThePickerOnTheFirstSpeciesThenAnEmptyNamingPrompt(t *testing.T) {
	for _, opts := range [][]option{{}, {monochrome()}} {
		h := newHarness(t, append(opts, named("Bubbles"), as(fish.Goldfish))...)
		h.resize(100, 24)
		h.advance(app.FrameInterval)
		h.release()
		h.advance(app.FrameInterval)

		screen := h.screen()
		if !strings.Contains(screen, "Species: ‹ Clownfish ›") {
			t.Fatalf("after y the picker doesn't open on the first Species:\n%s", screen)
		}
		if s, ok := findFish(h.rawView()); !ok || !s.intact {
			t.Errorf("the picker shows no whole clownfish:\n%s", screen)
		}
		if strings.Contains(screen, "Bubbles") {
			t.Errorf("the picker for a new Fish shows the old Fish's name:\n%s", screen)
		}

		h.press("enter")
		screen = h.screen()
		if !strings.Contains(screen, "What is your Fish called?") {
			t.Fatalf("choosing a Species didn't ask for a name:\n%s", screen)
		}
		if strings.Contains(screen, "Bubbles") || strings.TrimSpace(lineWith(screen, ">")) != ">" {
			t.Errorf("the naming prompt isn't empty:\n%s", screen)
		}
	}
}

// lineWith is the first line of screen holding s.
func lineWith(screen, s string) string {
	for _, l := range strings.Split(screen, "\n") {
		if strings.Contains(l, s) {
			return l
		}
	}
	return ""
}

func TestTheNewFishIsHatchedFreshAndSaved(t *testing.T) {
	h := inMood(t, "Starving")
	h.resize(100, 24)
	h.release()
	h.press("right") // a goldfish
	h.press("enter")
	if s := readSave(t, h.dataDir); s.Name != "Bubbles" {
		t.Fatalf("before the new Fish is named the save holds %q, want Bubbles", s.Name)
	}
	h.nameFish("Goldie")

	s := readSave(t, h.dataDir)
	if s.Name != "Goldie" || s.Species != "goldfish" || s.Version != 2 || !s.Born.Equal(h.clock.now) || s.Hunger != 0 {
		t.Errorf("after naming, the save is %+v, want a new goldfish Goldie born now with no Hunger", s)
	}
	if got, want := h.statusLine(), "Goldie · Hunger ▱▱▱▱▱▱▱▱▱▱ · Age 0 days"; !strings.HasPrefix(got, want) {
		t.Errorf("status line is %q, want it to start %q", got, want)
	}
	h.species = fish.Goldfish
	if mood := h.fishMood(); mood != "Content" {
		t.Errorf("the new Fish looks %q, want Content", mood)
	}

	again := h.reopen(time.Minute)
	again.resize(100, 24)
	again.advance(app.FrameInterval)
	if s, ok := findSpecies(again.rawView(), fish.Goldfish); !ok || !s.intact {
		t.Errorf("after a reopen the new Fish is not a whole goldfish:\n%s", again.screen())
	}
	if !strings.HasPrefix(again.statusLine(), "Goldie ·") {
		t.Errorf("after a reopen the status line is %q, want Goldie's", again.statusLine())
	}
}

func TestThePickerForANewFishDoesNotShowTheOldFishsMood(t *testing.T) {
	h := inMood(t, "Starving")
	h.resize(100, 24)
	h.release()
	h.advance(app.FrameInterval)

	if mood := h.fishMood(); mood != "Content" {
		t.Errorf("the picker for a new Fish shows a %q Fish, want Content", mood)
	}
}

func TestTheNewFishGetsNoGreeting(t *testing.T) {
	h := newHarness(t, named("Bubbles")).reopen(3 * time.Hour)
	h.resize(100, 24)
	if !strings.Contains(words(h.screen()), "missed you") {
		t.Fatalf("no greeting to check:\n%s", h.screen())
	}
	h.release()
	if strings.Contains(words(h.screen()), "missed you") {
		t.Errorf("the picker for a new Fish greets the user:\n%s", h.screen())
	}
	h.press("enter")
	h.nameFish("Nemo")
	if strings.Contains(words(h.screen()), "missed you") {
		t.Errorf("the new Fish's Tank greets the user:\n%s", h.screen())
	}
}

func TestTheNewFishStartsInAClearedTankWithTheSameScenery(t *testing.T) {
	h := newHarness(t, named("Bubbles")) // Full, so it leaves the Food alone
	h.resize(100, 24)
	for range 3 {
		h.press("f")
	}
	for i := 0; bubbles(h.view()) == 0; i++ {
		if i > 400 {
			t.Fatal("no bubble rose to check is cleared")
		}
		h.advance(app.FrameInterval)
	}
	if h.count(pellet) == 0 {
		t.Fatalf("no Food to check is cleared:\n%s", h.screen())
	}
	before := h.view()
	h.release()
	h.press("enter") // the clownfish again, so nothing needs regrowing
	h.nameFish("Nemo")

	after := h.view()
	if n := h.count(pellet); n != 0 {
		t.Errorf("the new Fish's Tank still holds %d pellets:\n%s", n, h.screen())
	}
	if n := bubbles(after); n != 0 {
		t.Errorf("the new Fish's Tank still holds %d bubbles:\n%s", n, h.screen())
	}
	floor := floorRow(before)
	if floorRow(after) != floor || after[floor] != before[floor] || after[floor-1] != before[floor-1] {
		t.Errorf("the sand or shells changed:\nbefore %q\n       %q\nafter  %q\n       %q", before[floor-1], before[floor], after[floor-1], after[floor])
	}
	if seaweedCells(after) != seaweedCells(before) {
		t.Errorf("the seaweed changed:\nbefore\n%s\nafter\n%s", strings.Join(before, "\n"), strings.Join(after, "\n"))
	}
}

func TestQuittingBeforeTheNewFishIsNamedKeepsTheOldFish(t *testing.T) {
	steps := map[string][]string{
		"picker esc":    {},
		"picker ctrl+c": {},
		"naming esc":    {"enter", "N", "e"},
		"naming ctrl+c": {"enter", "N", "e"},
	}
	for name, keys := range steps {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, named("Bubbles"))
			h.resize(100, 24)
			h.advance(30 * time.Second)
			saved, err := os.ReadFile(filepath.Join(h.dataDir, "fish.json"))
			if err != nil {
				t.Fatal(err)
			}
			h.release()
			h.advance(2 * time.Minute)
			for _, k := range keys {
				h.press(k)
			}
			h.press(strings.Fields(name)[1])
			if !h.quitRequested() {
				t.Fatalf("%s didn't quit", name)
			}
			if now, _ := os.ReadFile(filepath.Join(h.dataDir, "fish.json")); string(now) != string(saved) {
				t.Errorf("the save file changed after y:\nbefore %s\nafter  %s", saved, now)
			}

			again := newHarness(t, withDataDir(h.dataDir), at(h.clock.now.Add(time.Minute)))
			again.resize(100, 24)
			if !strings.HasPrefix(again.statusLine(), "Bubbles ·") {
				t.Errorf("after quitting, the next launch shows %q, want Bubbles' Tank", again.statusLine())
			}
		})
	}
}
