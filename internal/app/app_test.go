package app_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTankShowsFishAndSeaweedAboveSandyFloorAndStatusLine(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(40, 10)

	// The Fish starts in the middle of the water, its box 18 cells wide
	// and 4 rows tall, leaving 11 cells either side of it in 40.
	fish, ok := findFish(h.rawView())
	if !ok || !fish.intact || fish.left != 11 || fish.top != 2 {
		t.Fatalf("the Fish is at %+v, want whole with its box's top-left at (11, 2):\n%s", fish, h.screen())
	}
	want := []string{
		"",
		"",
		"",
		"",
		"",
		"",
		"                                       }",
		"                                      {",
		"░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░",
		" Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱ · • pellet",
	}
	got := h.view()
	if len(got) != len(want) {
		t.Fatalf("Tank has %d rows, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		// Everything but the Fish.
		row := cells(got[i])
		for x, c := range row {
			if isFishCell(c) {
				row[x] = " "
			}
		}
		if r := strings.TrimRight(strings.Join(row, ""), " "); r != want[i] {
			t.Errorf("row %d = %q without the Fish, want %q", i, r, want[i])
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
