package app_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestQuittingSavesTheFish(t *testing.T) {
	for _, key := range []string{"q", "esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			h := newHarness(t, named("Bubbles"))
			h.resize(50, 12)
			h.clock.now = h.clock.now.Add(6 * time.Hour) // the Tank stayed open 6 hours
			h.advance(time.Second)
			h.press(key)

			s := readSave(t, h.dataDir)
			if s.Hunger < 25 || s.Hunger > 25.01 {
				t.Errorf("saved Hunger = %v after 6 hours open, want 25", s.Hunger)
			}
			if !s.HungerUpdated.Equal(h.clock.now) {
				t.Errorf("saved hungerUpdated = %v, want the quit time %v", s.HungerUpdated, h.clock.now)
			}
			if !s.LastSeen.Equal(h.clock.now) {
				t.Errorf("saved lastSeen = %v, want the quit time %v", s.LastSeen, h.clock.now)
			}
		})
	}
}

func TestQuittingTheNamingPromptSavesNothing(t *testing.T) {
	h := newHarness(t)
	h.resize(50, 12)
	h.typeText("Bub")
	h.press("esc")

	if files := dataFiles(h.dataDir); len(files) != 0 {
		t.Errorf("quitting before naming saved files: %v", files)
	}
}

func TestTheFishIsSavedAboutOnceAMinuteWhileTheTankIsOpen(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(50, 12)

	h.advance(59 * time.Second)
	if s := readSave(t, h.dataDir); !s.HungerUpdated.Equal(start) {
		t.Errorf("saved after %v, before a minute was up", s.HungerUpdated.Sub(start))
	}

	h.advance(time.Second)
	s := readSave(t, h.dataDir)
	if !s.HungerUpdated.Equal(start.Add(time.Minute)) || !s.LastSeen.Equal(start.Add(time.Minute)) {
		t.Errorf("after a minute open the save is from %v (lastSeen %v), want %v", s.HungerUpdated, s.LastSeen, start.Add(time.Minute))
	}
	if s.Hunger <= 0 {
		t.Errorf("saved Hunger after a minute open = %v, want it risen above 0", s.Hunger)
	}

	h.advance(time.Minute)
	if s := readSave(t, h.dataDir); !s.HungerUpdated.Equal(start.Add(2 * time.Minute)) {
		t.Errorf("after two minutes open the save is from %v, want %v", s.HungerUpdated, start.Add(2*time.Minute))
	}
}

func TestABrokenSaveFileIsNotOverwrittenWhileOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fish.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, withDataDir(dir))
	h.resize(60, 10)
	h.advance(2 * time.Minute)
	h.press("q")

	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Errorf("the broken save file was overwritten with %q", raw)
	}
}
