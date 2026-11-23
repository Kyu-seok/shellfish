package app_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSecondTankInTheSameDataFolderIsRefused(t *testing.T) {
	first := newHarness(t, named("Bubbles"))
	before, err := os.ReadFile(filepath.Join(first.dataDir, "fish.json"))
	if err != nil {
		t.Fatal(err)
	}

	second := newHarness(t, withDataDir(first.dataDir))
	if !second.quitRequested() {
		t.Error("a refused launch does not quit")
	}
	second.resize(60, 10)

	screen := strings.Join(second.view(), "\n")
	if !strings.Contains(screen, "Bubbles is already swimming in another terminal") {
		t.Errorf("a second launch does not say Bubbles is already swimming:\n%s", screen)
	}
	if second.exitStatus() == 0 {
		t.Error("a refused launch exits with status 0, want non-zero")
	}
	second.press("q")
	second.advance(time.Second)
	if after, _ := os.ReadFile(filepath.Join(first.dataDir, "fish.json")); string(after) != string(before) {
		t.Errorf("a refused launch changed the save file:\n%s\nthen\n%s", before, after)
	}
	if first.exitStatus() != 0 {
		t.Error("the first Tank's exit status is non-zero")
	}
}

func TestSecondLaunchDuringNamingIsRefused(t *testing.T) {
	first := newHarness(t)
	first.resize(60, 10)

	second := newHarness(t, withDataDir(first.dataDir))
	second.resize(60, 10)

	screen := strings.Join(second.view(), "\n")
	if !strings.Contains(screen, "is already swimming in another terminal") {
		t.Errorf("a launch during naming is not refused:\n%s", screen)
	}
	if second.exitStatus() == 0 {
		t.Error("a refused launch exits with status 0, want non-zero")
	}
}

func TestLaunchSucceedsAfterTheFirstTankQuits(t *testing.T) {
	for _, key := range []string{"q", "esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			first := newHarness(t, named("Bubbles"))
			first.press(key)
			if !first.quitRequested() {
				t.Fatalf("%s does not quit", key)
			}

			again := newHarness(t, withDataDir(first.dataDir))
			again.resize(60, 10)
			screen := strings.Join(again.view(), "\n")
			if strings.Contains(screen, "already swimming") || again.exitStatus() != 0 {
				t.Errorf("a launch after quitting with %s was refused:\n%s", key, screen)
			}
			if !showsFish(screen) {
				t.Errorf("a launch after quitting with %s does not show the Fish:\n%s", key, screen)
			}
		})
	}
}

func TestLaunchSucceedsAfterQuittingTheNamingPrompt(t *testing.T) {
	first := newHarness(t)
	first.press("esc")

	again := newHarness(t, withDataDir(first.dataDir))
	again.resize(60, 10)
	if screen := strings.Join(again.view(), "\n"); !strings.Contains(screen, "What is your Fish called?") {
		t.Errorf("a launch after quitting the naming prompt was refused:\n%s", screen)
	}
}
