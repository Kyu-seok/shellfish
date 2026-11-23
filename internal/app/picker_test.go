package app_test

import (
	"strings"
	"testing"
)

func TestFirstLaunchShowsTheSpeciesPickerBeforeNaming(t *testing.T) {
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
