package app_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
	"github.com/Kyu-seok/shellfish/internal/fish"
)

func TestFirstLaunchShowsTheSpeciesPickerBeforeNaming(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.resize(60, 16)

	screen := h.screen()
	if strings.Contains(screen, "What is your Fish called?") {
		t.Fatalf("first launch asked for a name before a Species:\n%s", screen)
	}
	if !strings.Contains(screen, "Clownfish") {
		t.Errorf("the picker doesn't name the Species shown:\n%s", screen)
	}
	if !strings.Contains(screen, "←/→") || !strings.Contains(screen, "enter") {
		t.Errorf("the picker doesn't say which keys to use:\n%s", screen)
	}
	if s, ok := findFish(h.rawView()); !ok || !s.intact {
		t.Errorf("the picker shows no whole Fish swimming in the Tank:\n%s", screen)
	}
}

// goldfishBody is the goldfish's own body colour, which no clownfish has.
const goldfishBody = "#ffa630"

// fishColours is every colour the Fish is drawn in.
func (h *harness) fishColours() map[string]bool {
	seen := map[string]bool{}
	for _, c := range fishCells(h.view()) {
		st := h.styleAt(c[0], c[1])
		seen[st.fg], seen[st.bg] = true, true
	}
	delete(seen, "")
	return seen
}

func TestThePickerBrowsesTheSpeciesWithTheArrowKeys(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.resize(60, 16)

	// → steps through all ten Species in order, round to the first again,
	// and ← goes back round to the last.
	names := []string{"Clownfish", "Goldfish", "Pufferfish", "Blue tang", "Betta",
		"Angelfish", "Koi", "Guppy", "Neon tetra", "Anglerfish"}
	type step struct {
		key  string
		want int // index into names
	}
	var steps []step
	for i := 1; i <= len(names); i++ {
		steps = append(steps, step{"right", i % len(names)})
	}
	steps = append(steps, step{"left", len(names) - 1})
	for i, st := range steps {
		h.press(st.key)
		h.advance(app.FrameInterval)
		want := names[st.want]
		if !strings.Contains(h.screen(), "‹ "+want+" ›") {
			t.Fatalf("step %d: after %s the picker doesn't show %s:\n%s", i, st.key, want, h.screen())
		}
		sp := fish.AllSpecies[st.want]
		if s, ok := findSpecies(h.rawView(), sp); !ok || !s.intact {
			t.Errorf("step %d: no whole %s in the Tank:\n%s", i, want, h.screen())
		}
		if gold := hasColour(h.fishColours(), goldfishBody); gold != (sp == fish.Goldfish) {
			t.Errorf("step %d: showing %s, the Fish is drawn in %v", i, want, h.fishColours())
		}
	}
	if len(fish.AllSpecies) != len(names) {
		t.Errorf("the picker offers %d Species, want %d", len(fish.AllSpecies), len(names))
	}
}

func TestTheChosenSpeciesIsNamedSavedAndKept(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.resize(60, 16)
	h.press("right")
	h.press("enter")

	if screen := h.screen(); !strings.Contains(screen, "What is your Fish called?") {
		t.Fatalf("choosing a Species didn't ask for a name:\n%s", screen)
	}
	if files := dataFiles(h.dataDir); len(files) != 0 {
		t.Errorf("choosing a Species for a new Fish saved %v before it was named", files)
	}
	h.nameFish("Goldie")
	if s := readSave(t, h.dataDir); s.Species != "goldfish" || s.Version != 2 {
		t.Errorf("saved version %d species %q, want version 2 goldfish", s.Version, s.Species)
	}

	again := h.reopen(time.Hour)
	again.resize(60, 16)
	again.advance(app.FrameInterval)
	if s, ok := findSpecies(again.rawView(), fish.Goldfish); !ok || !s.intact || !hasColour(again.fishColours(), goldfishBody) {
		t.Errorf("after a reopen the Fish is not a whole goldfish:\n%s", again.screen())
	}
	if strings.Contains(again.screen(), "Species: ‹") {
		t.Errorf("a reopen showed the picker again:\n%s", again.screen())
	}
}

func TestEscAndCtrlCQuitThePickerWithoutSaving(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			h := newHarness(t)
			h.resize(60, 16)
			h.press("right")
			h.press(key)

			if !h.quitRequested() {
				t.Errorf("%s does not quit the picker", key)
			}
			if files := dataFiles(h.dataDir); len(files) != 0 {
				t.Errorf("quitting the picker saved %v", files)
			}
		})
	}
}

func TestQIsNotSpecialOnThePicker(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.resize(60, 16)
	h.press("q")

	if h.quitRequested() {
		t.Error("q quit the picker")
	}
	if !strings.Contains(h.screen(), "Species: ‹ Clownfish ›") {
		t.Errorf("q left the picker:\n%s", h.screen())
	}
}

// writeVersion1Save writes a Fish as a shellfish from before Species saved
// it: schema version 1, with a pastel colour and no Species.
func writeVersion1Save(t *testing.T, dir string, born, lastSeen time.Time, hunger float64) {
	t.Helper()
	raw := fmt.Sprintf(`{
  "version": 1,
  "name": "Bubbles",
  "colour": "#BAE1FF",
  "born": %q,
  "hunger": %v,
  "hungerUpdated": %q,
  "lastSeen": %q
}
`, born.Format(time.RFC3339), hunger, lastSeen.Format(time.RFC3339), lastSeen.Format(time.RFC3339))
	if err := os.WriteFile(filepath.Join(dir, "fish.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAVersion1FishPicksItsSpeciesOnceAndKeepsEverythingElse(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	born := start.Add(-3 * 24 * time.Hour)
	lastSeen := start.Add(-6 * time.Hour)
	writeVersion1Save(t, dir, born, lastSeen, 10)

	h := newHarness(t, withDataDir(dir))
	h.resize(60, 16)
	if screen := h.screen(); !strings.Contains(screen, "Species: ‹ Clownfish ›") || strings.Contains(screen, "What is your Fish called?") {
		t.Fatalf("a version-1 Fish didn't open the picker:\n%s", screen)
	}
	if s := readSave(t, dir); s.Version != 1 {
		t.Errorf("the picker rewrote the save before a Species was chosen: version %d", s.Version)
	}
	h.press("right")
	h.press("enter")

	screen := h.screen()
	if !strings.Contains(screen, "Bubbles missed you! You were gone 6 hours.") {
		t.Errorf("no welcome-back greeting after picking:\n%s", screen)
	}
	if !strings.Contains(screen, "Age 3 days") {
		t.Errorf("the Fish's age was not kept:\n%s", screen)
	}
	if s, ok := findSpecies(h.rawView(), fish.Goldfish); !ok || !s.intact {
		t.Errorf("the Fish is not drawn as the goldfish picked:\n%s", screen)
	}
	s := readSave(t, dir)
	if s.Version != 2 || s.Species != "goldfish" || s.Name != "Bubbles" || !s.Born.Equal(born) {
		t.Errorf("saved %+v, want version 2, goldfish, Bubbles born %v", s, born)
	}
	// Hunger kept counting while the app was closed: 10, plus a quarter of
	// a Day's worth.
	if want := 10 + 0.25*100.0; s.Hunger < want-0.5 || s.Hunger > want+0.5 {
		t.Errorf("Hunger = %v, want about %v", s.Hunger, want)
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "fish.json")); strings.Contains(string(raw), "colour") {
		t.Errorf("the old colour is still saved:\n%s", raw)
	}

	again := h.reopen(time.Minute)
	again.resize(60, 16)
	if screen := again.screen(); strings.Contains(screen, "Species: ‹") {
		t.Errorf("the picker was shown again:\n%s", screen)
	}
	if s, ok := findSpecies(again.rawView(), fish.Goldfish); !ok || !s.intact {
		t.Errorf("after a reopen the Fish is not a goldfish:\n%s", again.screen())
	}
}

func TestAnUnknownSaveVersionIsRefused(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	raw := `{"version": 3, "name": "Bubbles", "species": "clownfish"}`
	if err := os.WriteFile(filepath.Join(dir, "fish.json"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, withDataDir(dir))
	h.resize(80, 10)

	screen := h.screen()
	if !strings.Contains(screen, "schema version 3") {
		t.Errorf("an unknown save version is not reported:\n%s", screen)
	}
	if strings.Contains(screen, "Species: ‹") || showsFish(screen) {
		t.Errorf("an unknown save version opened anyway:\n%s", screen)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "fish.json")); string(got) != raw {
		t.Errorf("the save file was overwritten with %q", got)
	}
}
