package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
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

// blockSaves makes every save to h's data folder fail, by putting a folder
// where the save file goes, and returns a func that lets saves through again.
func (h *harness) blockSaves() (unblock func()) {
	h.t.Helper()
	path := filepath.Join(h.dataDir, "fish.json")
	if err := os.Remove(path); err != nil {
		h.t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "in-the-way"), 0o755); err != nil {
		h.t.Fatal(err)
	}
	return func() {
		if err := os.RemoveAll(path); err != nil {
			h.t.Fatal(err)
		}
	}
}

func TestAFailedSaveKeepsTheTankOpenAndIsRetried(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)
	unblock := h.blockSaves()

	h.advance(time.Minute)
	if h.quitRequested() || !strings.Contains(h.screen(), contentFace) {
		t.Fatalf("a failed save closed the Tank:\n%s", h.screen())
	}
	if !strings.Contains(h.statusLine(), "couldn't save") {
		t.Errorf("status line after a failed save is %q, want it to say the Fish couldn't be saved", h.statusLine())
	}

	h.clock.now = h.clock.now.Add(12 * time.Hour) // Hunger keeps rising
	h.advance(time.Second)
	if !strings.Contains(h.screen(), hungryFace) {
		t.Errorf("12 hours after a failed save the Fish is not Hungry:\n%s", h.screen())
	}

	unblock()
	h.advance(time.Minute)
	if s := readSave(t, h.dataDir); s.Hunger < 50 {
		t.Errorf("the next periodic save wrote Hunger %v, want at least 50", s.Hunger)
	}
	if strings.Contains(h.statusLine(), "couldn't save") {
		t.Errorf("status line still reports a failed save after saving again: %q", h.statusLine())
	}
}

func TestAFailedSaveOnQuitExitsWithTheError(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)
	h.blockSaves()

	h.press("q")
	if !h.quitRequested() {
		t.Fatal("q does not quit when the save fails")
	}
	if h.exitStatus() == 0 {
		t.Error("the exit status is 0 after the quit save failed")
	}
	if msg := h.model.(app.Model).ExitMessage(); !strings.Contains(msg, "saving Fish") {
		t.Errorf("exit message is %q, want the save error", msg)
	}
}
