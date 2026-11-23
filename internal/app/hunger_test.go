package app_test

import (
	"strings"
	"testing"
	"time"
)

// statusLine is the bottom row of the Tank, trimmed.
func (h *harness) statusLine() string {
	got := h.view()
	return strings.TrimSpace(got[len(got)-1])
}

// screen is the whole rendered Tank as plain text.
func (h *harness) screen() string { return strings.Join(h.view(), "\n") }

// reopen launches the app again on h's data folder with the clock moved by d,
// as if the user had quit and come back d later.
func (h *harness) reopen(d time.Duration) *harness {
	h.t.Helper()
	return newHarness(h.t, withDataDir(h.dataDir), at(h.clock.now.Add(d)))
}

const (
	contentFace  = "◕ ᴗ"
	hungryFace   = "• _"
	starvingFace = "; ﹏"
)

func TestHungerCatchesUpOnTimeAway(t *testing.T) {
	for _, tc := range []struct {
		name  string
		away  time.Duration
		face  string
		meter string
	}{
		{"12 hours", 12 * time.Hour, hungryFace, "Hunger ▰▰▰▰▰▱▱▱▱▱"},
		{"26 hours, capped at 100", 26 * time.Hour, starvingFace, "Hunger ▰▰▰▰▰▰▰▰▰▰"},
		{"clock moved back 5 hours", -5 * time.Hour, contentFace, "Hunger ▱▱▱▱▱▱▱▱▱▱"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := newHarness(t, named("Bubbles"))
			h := first.reopen(tc.away)
			h.resize(50, 12)

			if !strings.Contains(h.screen(), tc.face) {
				t.Errorf("after %v away the Fish's face is not %q:\n%s", tc.away, tc.face, h.screen())
			}
			if !strings.Contains(h.statusLine(), tc.meter) {
				t.Errorf("after %v away the status line is %q, want it to show %q", tc.away, h.statusLine(), tc.meter)
			}
		})
	}
}

func TestFaceShowsTheMoodBand(t *testing.T) {
	// Hunger rises 100 per 24 hours, so 9h36m is Hunger 40 and 19h12m is 80.
	for _, tc := range []struct {
		away time.Duration
		row  string
	}{
		{0, "><(  ◕ ᴗ )"},
		{9*time.Hour + 35*time.Minute, "><(  ◕ ᴗ )"},
		{9*time.Hour + 36*time.Minute, "><(  • _ )"},
		{19*time.Hour + 11*time.Minute, "><(  • _ )"},
		{19*time.Hour + 12*time.Minute, "><(  ; ﹏ )"},
		{30 * 24 * time.Hour, "><(  ; ﹏ )"},
	} {
		first := newHarness(t, named("Bubbles"))
		h := first.reopen(tc.away)
		h.resize(50, 12)

		found := false
		for _, row := range h.view() {
			if strings.TrimSpace(row) == tc.row {
				found = true
			}
		}
		if !found {
			t.Errorf("after %v away no row reads %q:\n%s", tc.away, tc.row, h.screen())
		}
	}
}

func TestHungerRisesWhileTheTankIsOpen(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(50, 12)
	h.clock.now = h.clock.now.Add(12 * time.Hour) // as if the Tank had stayed open half a day
	h.advance(time.Second)

	if !strings.Contains(h.screen(), hungryFace) {
		t.Errorf("after 12 hours open the Fish is not Hungry:\n%s", h.screen())
	}
	if !strings.Contains(h.statusLine(), "Hunger ▰▰▰▰▰▱▱▱▱▱") {
		t.Errorf("after 12 hours open the status line is %q", h.statusLine())
	}
}
