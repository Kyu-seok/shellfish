package app_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestTankShowsFishAboveSandyFloor(t *testing.T) {
	h := newHarness(t)
	h.resize(30, 10)

	want := []string{
		"",
		"",
		"",
		"             .-~~-.",
		"          ><(  ◕ ᴗ )",
		"             `-..-'",
		"",
		"",
		"",
		"░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░",
	}
	got := h.view()
	if len(got) != len(want) {
		t.Fatalf("Tank has %d rows, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if strings.TrimRight(got[i], " ") != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTankFillsTheWindowAndFollowsResizes(t *testing.T) {
	h := newHarness(t)
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
		if last := got[len(got)-1]; !strings.Contains(last, "░░░░") {
			t.Errorf("%dx%d: bottom row %q is not sand", size.w, size.h, last)
		}
	}
}

func TestQuitKeysEndTheProgram(t *testing.T) {
	for _, key := range []string{"q", "esc", "ctrl+c"} {
		t.Run(key, func(t *testing.T) {
			h := newHarness(t)
			h.resize(80, 24)
			h.press(key)
			if !h.quitRequested() {
				t.Errorf("pressing %s did not quit", key)
			}
		})
	}
}

func TestOtherKeysDoNotQuit(t *testing.T) {
	h := newHarness(t)
	h.resize(80, 24)
	h.press("x")
	if h.quitRequested() {
		t.Error("pressing x quit the program")
	}
}
