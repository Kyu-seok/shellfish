package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// savedFish is the save file as the spec describes it.
type savedFish struct {
	Version       int       `json:"version"`
	Name          string    `json:"name"`
	Colour        string    `json:"colour"`
	Born          time.Time `json:"born"`
	Hunger        float64   `json:"hunger"`
	HungerUpdated time.Time `json:"hungerUpdated"`
	LastSeen      time.Time `json:"lastSeen"`
}

// dataFiles lists the data folder's entries, leaving out the lock file that
// an open Tank holds.
func dataFiles(dataDir string) []string {
	entries, _ := os.ReadDir(dataDir)
	var names []string
	for _, e := range entries {
		if e.Name() != "fish.lock" {
			names = append(names, e.Name())
		}
	}
	return names
}

func readSave(t *testing.T, dataDir string) savedFish {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dataDir, "fish.json"))
	if err != nil {
		t.Fatalf("reading save file: %v", err)
	}
	var s savedFish
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("save file is not JSON: %v\n%s", err, raw)
	}
	return s
}

func TestNamingSavesAFullFishWithAPastelColour(t *testing.T) {
	h := newHarness(t)
	h.resize(40, 10)
	h.nameFish("  Bubbles ")

	s := readSave(t, h.dataDir)
	if s.Version != 1 {
		t.Errorf("schema version = %d, want 1", s.Version)
	}
	if s.Name != "Bubbles" {
		t.Errorf("name = %q, want %q", s.Name, "Bubbles")
	}
	if !regexp.MustCompile(`^#[0-9A-F]{6}$`).MatchString(s.Colour) {
		t.Errorf("colour = %q, want a #RRGGBB colour", s.Colour)
	}
	if s.Hunger != 0 {
		t.Errorf("Hunger = %v, want 0 (Full)", s.Hunger)
	}
	for field, got := range map[string]time.Time{"born": s.Born, "hungerUpdated": s.HungerUpdated, "lastSeen": s.LastSeen} {
		if !got.Equal(start) {
			t.Errorf("%s = %v, want the naming time %v", field, got, start)
		}
	}
	if files := dataFiles(h.dataDir); len(files) != 1 {
		t.Errorf("data folder holds %v, want only the save file (no leftover temporary file)", files)
	}
}

func TestNamingCreatesAMissingDataFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "shellfish")
	h := newHarness(t, withDataDir(dir))
	h.resize(40, 10)
	h.nameFish("Bubbles")

	if s := readSave(t, dir); s.Name != "Bubbles" {
		t.Errorf("name = %q, want Bubbles", s.Name)
	}
}

func TestLaterLaunchLoadsTheSameFish(t *testing.T) {
	first := newHarness(t, named("Bubbles"))
	saved := readSave(t, first.dataDir)
	first.press("q")

	again := newHarness(t, withDataDir(first.dataDir), at(start.Add(time.Hour)))
	again.resize(40, 10)

	screen := strings.Join(again.view(), "\n")
	if strings.Contains(screen, "What is your Fish called?") {
		t.Fatalf("a later launch asked for a name again:\n%s", screen)
	}
	if !strings.Contains(screen, "Bubbles") || !showsFish(screen) {
		t.Errorf("a later launch does not show Bubbles in the Tank:\n%s", screen)
	}
	if got := readSave(t, first.dataDir); got.Colour != saved.Colour || !got.Born.Equal(saved.Born) {
		t.Errorf("the Fish changed between launches: %+v, then %+v", saved, got)
	}
}

func TestUnreadableSaveFileIsReportedNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fish.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, withDataDir(dir))
	h.resize(60, 10)

	screen := strings.Join(h.view(), "\n")
	if strings.Contains(screen, "What is your Fish called?") {
		t.Errorf("a broken save file was treated as a first launch:\n%s", screen)
	}
	if !strings.Contains(screen, "saved Fish") {
		t.Errorf("the broken save file is not reported:\n%s", screen)
	}
	if raw, _ := os.ReadFile(path); string(raw) != "{not json" {
		t.Errorf("the broken save file was overwritten with %q", raw)
	}
	h.press("q")
	if !h.quitRequested() {
		t.Error("q does not quit from the error screen")
	}
}
