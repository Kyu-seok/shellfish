package app_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

const bigger = "Make me a bigger tank 🐟"

// words is the screen's text with all line breaks and padding squeezed to
// single spaces, so a message reads the same however it wraps.
func words(screen string) string { return strings.Join(strings.Fields(screen), " ") }

func TestTooSmallWindowAsksForABiggerTank(t *testing.T) {
	for _, size := range []struct{ w, h int }{{29, 10}, {30, 9}, {20, 5}} {
		h := newHarness(t, named("Bubbles"))
		h.resize(size.w, size.h)

		screen := strings.Join(h.view(), "\n")
		if !strings.Contains(words(screen), bigger) {
			t.Errorf("%dx%d: no request for a bigger tank:\n%s", size.w, size.h, screen)
		}
		if showsFish(h.screen()) || strings.Contains(screen, "░") {
			t.Errorf("%dx%d: a broken Tank was drawn:\n%s", size.w, size.h, screen)
		}
	}
}

func TestTooSmallMessageIsCentred(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(29, 5)

	want := []string{
		"",
		"",
		"  " + bigger,
		"",
		"",
	}
	got := h.view()
	if len(got) != len(want) {
		t.Fatalf("screen has %d rows, want %d:\n%s", len(got), len(want), strings.Join(got, "\n"))
	}
	for i := range want {
		if strings.TrimRight(got[i], " ") != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTankReturnsAfterResizingAboveTheMinimum(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(20, 5)
	h.resize(30, 10)

	screen := strings.Join(h.view(), "\n")
	if strings.Contains(words(screen), bigger) {
		t.Errorf("still asking for a bigger tank at 30x10:\n%s", screen)
	}
	if !showsFish(h.screen()) {
		t.Errorf("the Tank did not come back:\n%s", screen)
	}
}

// Hunger and Age both carry on advancing while the Tank can't be drawn.
func TestLifeCarriesOnWhileTheWindowIsTooSmall(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(20, 5)
	h.advance(12 * time.Hour)

	if screen := strings.Join(h.view(), "\n"); !strings.Contains(words(screen), bigger) {
		t.Fatalf("the Tank was drawn in a too-small window:\n%s", screen)
	}
	h.resize(60, 10)
	got := h.view()
	if status := got[len(got)-1]; !strings.Contains(status, "Hunger ▰▰▰▰▰▱▱▱▱▱") {
		t.Errorf("after half a day in a too-small window Hunger has not advanced: %q", status)
	}
	h.resize(20, 5)
	h.advance(12 * time.Hour)
	h.resize(60, 10)
	got = h.view()
	if status := got[len(got)-1]; !strings.Contains(status, "Hunger ▰▰▰▰▰▰▰▰▰▰") || !strings.Contains(status, "Age 1 day") {
		t.Errorf("after a day in a too-small window the status line is %q", status)
	}
	if h.quitRequested() {
		t.Error("the app quit while the window was too small")
	}
}

// colourSGR matches a foreground or background colour escape code.
var colourSGR = regexp.MustCompile(`\x1b\[([0-9]+;)*(3[0-9]|4[0-9]|9[0-7]|10[0-7])[;m]`)

func TestMonochromeDrawsNoColour(t *testing.T) {
	h := newHarness(t, named("Bubbles"), monochrome())
	h.resize(80, 24)
	for i, key := range []string{"1", "2", "3"} { // a pellet, a flake and a worm
		h.press(key)
		h.click(30+5*i, 2)
	}

	// Watch the Food fall and bubbles rise, with the Fish found and whole
	// in every frame.
	bubble := false
	for i := range int(10 * time.Second / app.FrameInterval) {
		h.watch(app.FrameInterval)
		raw := strings.Join(h.rawView(), "\n")
		if colourSGR.MatchString(raw) {
			t.Fatalf("frame %d: monochrome Tank has colour escape codes: %q", i, raw)
		}
		rows := h.view()
		for _, row := range rows[waterTop:floorRow(rows)] {
			bubble = bubble || strings.ContainsAny(row, "°oO")
		}
	}
	if !bubble {
		t.Error("no bubble rose in 10 seconds to check for colour")
	}
	screen := strings.Join(h.view(), "\n")
	if !strings.Contains(screen, "Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱ · Age 0 days") {
		t.Errorf("monochrome Tank is missing the status line:\n%s", screen)
	}
	h.press("q")
	if !h.quitRequested() {
		t.Error("q does not quit in monochrome")
	}
}

// In monochrome the Fish is a plain silhouette, with its eye left clear so
// it still reads as a face.
func TestMonochromeFishHasAHoleForItsEye(t *testing.T) {
	h := newHarness(t, named("Bubbles"), monochrome())
	h.resize(80, 24)
	turned := map[bool]bool{}
	for i := range int(20 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		rows := h.view()
		s, ok := findFish(h.rawView())
		if !ok || !s.intact {
			t.Fatalf("frame %d: no whole Fish found in monochrome:\n%s", i, strings.Join(rows, "\n"))
		}
		if !eyeHole(rows, art, s.left, s.top, s.facingLeft) {
			t.Fatalf("frame %d: the Fish's eye is not a hole in its silhouette:\n%s", i, strings.Join(rows, "\n"))
		}
		turned[s.facingLeft] = true
	}
	if len(turned) < 2 {
		t.Error("the Fish never turned in 20 seconds, so only one way was checked")
	}
}

// Fading things are still faint in monochrome: faint is not a colour.
func TestFadingFoodIsFaintInMonochrome(t *testing.T) {
	h := newHarness(t, named("Bubbles"), monochrome()) // Full, so it leaves the Food alone
	h.resize(80, 24)
	h.click(5, h.bottom()) // dropped on the sand, so it rests at once

	for range int(30 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		if h.cellAt(5, h.bottom()) == pellet && h.styleAt(5, h.bottom()).faint {
			return
		}
	}
	t.Error("the pellet never faded in monochrome")
}

func TestColourTankDoesDrawColour(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)
	if !colourSGR.MatchString(strings.Join(h.rawView(), "\n")) {
		t.Error("the colour Tank has no colour escape codes, so the monochrome test proves nothing")
	}
}

const keyHint = "click/f feed · 1-3 food · r release · q quit"

func TestStatusLineEndsWithAFaintKeyHint(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(100, 24)

	got := h.view()
	status := got[len(got)-1]
	if !strings.HasSuffix(strings.TrimRight(status, " "), keyHint) {
		t.Fatalf("status line %q does not end with the key hint", status)
	}
	if strings.TrimRight(status, " ") == status {
		t.Errorf("key hint is flush against the edge: %q", status)
	}
	if len([]rune(status))-len([]rune(strings.TrimRight(status, " "))) != 1 {
		t.Errorf("key hint is not at the right end of the status line: %q", status)
	}
	row := len(got) - 1
	hint := len([]rune(strings.TrimRight(status, " "))) - len([]rune(keyHint))
	if !h.styleAt(hint, row).faint {
		t.Errorf("key hint is not faint: %q", h.rawView()[row])
	}
	if h.styleAt(0, row).faint {
		t.Errorf("the Fish's name is faint too: %q", h.rawView()[row])
	}
}

func TestKeyHintMakesWayInANarrowTank(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	for _, tc := range []struct {
		width int
		want  string
	}{
		{50, "Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱ · Age 0 days · •"},
		{40, "Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱ · • pellet"},
		{30, "Bubbles · Hunger ▱▱▱▱▱▱▱▱▱▱"},
	} {
		h.resize(tc.width, 10)
		got := h.view()
		if status := strings.TrimSpace(got[len(got)-1]); status != tc.want {
			t.Errorf("%d wide: status line is %q, want %q without a clipped hint or a dangling ·", tc.width, status, tc.want)
		}
	}
}
