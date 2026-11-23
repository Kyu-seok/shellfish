package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/Kyu-seok/shellfish/internal/fish"
)

// greeting welcomes the user back to f at now, or is empty if they were away
// GreetAfter or less.
func greeting(f fish.Fish, now time.Time) string {
	if f.LastSeen.IsZero() {
		return ""
	}
	away := now.Sub(f.LastSeen)
	if away <= GreetAfter {
		return ""
	}
	return fmt.Sprintf("%s missed you! You were gone %s.", f.Name, readable(away))
}

// readable is an absence in whole hours, or whole days from a day on.
func readable(d time.Duration) string {
	n, unit := int(d/time.Hour), "hour"
	if d >= 24*time.Hour {
		n, unit = int(d/(24*time.Hour)), "day"
	}
	if n != 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s", n, unit)
}

// drawGreeting overlays the welcome-back greeting, centred and wrapped to fit,
// near the top of the Tank's water rows while it is still up.
func (m Model) drawGreeting(water []string) {
	if m.greeting == "" || !m.now.Before(m.greetUntil) {
		return
	}
	lines := strings.Split(ansi.Wordwrap(m.greeting, m.width-2, ""), "\n")
	for i, line := range lines {
		y := 1 + i
		if y >= len(water) {
			return
		}
		left := max((m.width-ansi.StringWidth(line))/2, 0)
		water[y] = fit(strings.Repeat(" ", left)+line, m.width)
	}
}
