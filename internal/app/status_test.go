package app_test

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestStatusLineShowsNameAndAgeInDays(t *testing.T) {
	first := newHarness(t, named("Bubbles"))
	first.press("q")

	for _, tc := range []struct {
		after time.Duration
		want  string
	}{
		{0, "Age 0 days"},
		{23 * time.Hour, "Age 0 days"},
		{24 * time.Hour, "Age 1 day"},
		{3*24*time.Hour + 5*time.Hour, "Age 3 days"},
	} {
		h := newHarness(t, withDataDir(first.dataDir), at(start.Add(tc.after)))
		h.resize(60, 10)
		got := h.view()
		if status := strings.TrimSpace(got[len(got)-1]); !strings.HasPrefix(status, "Bubbles · ") || ageShown(status) != tc.want {
			t.Errorf("after %v the status line is %q, want the name first and %q", tc.after, status, tc.want)
		}
		h.press("q")
	}
}

func TestAgeTicksOverWhileTheTankIsOpen(t *testing.T) {
	h := newHarness(t, named("Bubbles"), at(start.Add(-24*time.Hour+time.Second)))
	h.resize(60, 10)
	h.clock.now = start // as if the Tank had stayed open almost a day
	h.advance(2 * time.Second)

	got := h.view()
	if status := strings.TrimSpace(got[len(got)-1]); ageShown(status) != "Age 1 day" {
		t.Errorf("status line is %q after the Fish turned one day old", status)
	}
}

// ageShown is the "Age N days" item on a status line, or "" if there is none.
func ageShown(status string) string {
	return regexp.MustCompile(`Age \d+ days?\b`).FindString(status)
}
