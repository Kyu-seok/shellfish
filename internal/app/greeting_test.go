package app_test

import (
	"strings"
	"testing"
	"time"
)

func TestComingBackAfterTimeAwayGreetsTheUser(t *testing.T) {
	for _, tc := range []struct {
		away time.Duration
		want string
	}{
		{3 * time.Hour, "Bubbles missed you! You were gone 3 hours."},
		{2 * 24 * time.Hour, "Bubbles missed you! You were gone 2 days."},
		{90 * time.Minute, "Bubbles missed you! You were gone 1 hour."},
		{24 * time.Hour, "Bubbles missed you! You were gone 1 day."},
	} {
		t.Run(tc.away.String(), func(t *testing.T) {
			h := newHarness(t, named("Bubbles")).reopen(tc.away)
			h.resize(60, 14)

			if got := words(h.screen()); !strings.Contains(got, tc.want) {
				t.Errorf("Tank after %v away = %q, want it to say %q", tc.away, got, tc.want)
			}
		})
	}
}

func TestShortAbsencesGetNoGreeting(t *testing.T) {
	for _, away := range []time.Duration{30 * time.Minute, time.Hour} {
		t.Run(away.String(), func(t *testing.T) {
			h := newHarness(t, named("Bubbles")).reopen(away)
			h.resize(60, 14)

			if got := h.screen(); strings.Contains(got, "missed you") {
				t.Errorf("Tank after %v away greets the user:\n%s", away, got)
			}
		})
	}
}

func TestTheFirstLaunchGetsNoGreeting(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(60, 14)

	if got := h.screen(); strings.Contains(got, "missed you") {
		t.Errorf("the first launch greets the user:\n%s", got)
	}
}

func TestTheGreetingGoesAwayByItself(t *testing.T) {
	h := newHarness(t, named("Bubbles")).reopen(3 * time.Hour)
	h.resize(60, 14)

	h.advance(4 * time.Second)
	if !strings.Contains(h.screen(), "missed you") {
		t.Fatalf("the greeting was gone after 4 seconds, want it still up")
	}
	h.advance(time.Second)
	if got := h.screen(); strings.Contains(got, "missed you") {
		t.Errorf("the greeting is still up after 5 seconds:\n%s", got)
	}
}

func TestTheGreetingIsDrawnNearTheTopOfTheTank(t *testing.T) {
	h := newHarness(t, named("Bubbles")).reopen(3 * time.Hour)
	h.resize(60, 14)

	// Just below the top row of water, where f drops Food.
	rows := h.view()
	if !strings.Contains(rows[waterTop+1], "Bubbles missed you!") {
		t.Errorf("row %d = %q, want the greeting there:\n%s", waterTop+1, rows[waterTop+1], h.screen())
	}
	if h.fishMood() != "Content" {
		t.Errorf("the greeting hides the Fish:\n%s", h.screen())
	}
}

func TestTheGreetingWrapsInANarrowTank(t *testing.T) {
	h := newHarness(t, named("Bubbles")).reopen(3 * time.Hour)
	h.resize(30, 12)

	for _, row := range h.view() {
		if w := len([]rune(row)); w != 30 {
			t.Errorf("row %q is %d cells wide, want 30", row, w)
		}
	}
	if got, want := words(h.screen()), "Bubbles missed you! You were gone 3 hours."; !strings.Contains(got, want) {
		t.Errorf("narrow Tank = %q, want the whole greeting %q", got, want)
	}
}
