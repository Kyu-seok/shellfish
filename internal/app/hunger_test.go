package app_test

import (
	"strings"
	"testing"
	"time"
)

func TestHungerCatchesUpOnTimeAway(t *testing.T) {
	for _, tc := range []struct {
		name  string
		away  time.Duration
		mood  string
		meter string
	}{
		{"12 hours", 12 * time.Hour, "Hungry", "Hunger ▰▰▰▰▰▱▱▱▱▱"},
		{"26 hours, capped at 100", 26 * time.Hour, "Starving", "Hunger ▰▰▰▰▰▰▰▰▰▰"},
		{"clock moved back 5 hours", -5 * time.Hour, "Content", "Hunger ▱▱▱▱▱▱▱▱▱▱"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := newHarness(t, named("Bubbles"))
			h := first.reopen(tc.away)
			h.resize(50, 12)

			if got := h.fishMood(); got != tc.mood {
				t.Errorf("after %v away the Fish's eye looks %q, want %q:\n%s", tc.away, got, tc.mood, h.screen())
			}
			if !strings.Contains(h.statusLine(), tc.meter) {
				t.Errorf("after %v away the status line is %q, want it to show %q", tc.away, h.statusLine(), tc.meter)
			}
		})
	}
}

// The eye is wide open and white when Content, half-shut by its lid when
// Hungry, and shut further with a tear below it when Starving.
func TestTheEyeShowsTheMoodBand(t *testing.T) {
	// Hunger rises 100 per 24 hours, so 9h36m is Hunger 40 and 19h12m is 80.
	for _, tc := range []struct {
		away time.Duration
		mood string
	}{
		{0, "Content"},
		{9*time.Hour + 35*time.Minute, "Content"},
		{9*time.Hour + 36*time.Minute, "Hungry"},
		{19*time.Hour + 11*time.Minute, "Hungry"},
		{19*time.Hour + 12*time.Minute, "Starving"},
		{30 * 24 * time.Hour, "Starving"},
	} {
		first := newHarness(t, named("Bubbles"))
		h := first.reopen(tc.away)
		h.resize(50, 12)

		if got := h.fishMood(); got != tc.mood {
			t.Errorf("after %v away the Fish's eye looks %q, want %q:\n%s", tc.away, got, tc.mood, h.screen())
		}
	}
}

func TestHungerRisesWhileTheTankIsOpen(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(50, 12)
	h.clock.now = h.clock.now.Add(12 * time.Hour) // as if the Tank had stayed open half a day
	h.advance(time.Second)

	if h.fishMood() != "Hungry" {
		t.Errorf("after 12 hours open the Fish is not Hungry:\n%s", h.screen())
	}
	if !strings.Contains(h.statusLine(), "Hunger ▰▰▰▰▰▱▱▱▱▱") {
		t.Errorf("after 12 hours open the status line is %q", h.statusLine())
	}
}
