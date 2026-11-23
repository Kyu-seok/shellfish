package app_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTankShowsFishAndSeaweedAboveSandyFloorAndStatusLine(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(30, 10)

	want := []string{
		"", // the surface
		"",
		"             .-~~-.",
		"          ><(  ◕ ᴗ )",
		"             `-..-'",
		"                            {",
		"                             }",
		"                            {",
		"░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░",
		" Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱",
	}
	got := h.view()
	if len(got) != len(want) {
		t.Fatalf("Tank has %d rows, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	if !isWave(got[0]) {
		t.Errorf("row 0 = %q, want the surface's wave line", got[0])
	}
	for i := 1; i < len(want); i++ {
		if strings.TrimRight(got[i], " ") != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTankFillsTheWindowAndFollowsResizes(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	for _, size := range []struct{ w, h int }{{30, 10}, {80, 24}, {120, 40}} {
		h.resize(size.w, size.h)
		got := h.view()
		if len(got) != size.h {
			t.Errorf("%dx%d: Tank has %d rows", size.w, size.h, len(got))
		}
		for i, row := range got {
			if w := ansi.StringWidth(row); w != size.w {
				t.Errorf("%dx%d: row %d is %d cells wide", size.w, size.h, i, w)
			}
		}
		if floor := got[len(got)-2]; !strings.Contains(floor, "░░░░") {
			t.Errorf("%dx%d: row above the status line %q is not sand", size.w, size.h, floor)
		}
	}
}

func TestQuitKeysEndTheProgram(t *testing.T) {
	for _, key := range []string{"q", "esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			h := newHarness(t, named("Bubbles"))
			h.resize(80, 24)
			h.press(key)
			if !h.quitRequested() {
				t.Errorf("pressing %s did not quit", key)
			}
		})
	}
}

func TestOtherKeysDoNotQuit(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)
	h.press("x")
	if h.quitRequested() {
		t.Error("pressing x quit the program")
	}
}
