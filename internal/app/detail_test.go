package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

func TestDDrawsTheFishInCoarseDetailInTheSameCells(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)
	x0, y0, x1, y1, ok := fishBounds(h.view())
	if !ok {
		t.Fatalf("no Fish in the Tank:\n%s", h.screen())
	}

	h.press("d")

	rows := h.view()
	quadrants := 0
	for _, c := range fishCells(rows) {
		cell := cells(rows[c[1]])[c[0]]
		if isSextant(cell) {
			t.Fatalf("cell (%d, %d) of a Coarse Fish is the sextant %q:\n%s", c[0], c[1], cell, h.screen())
		}
		if cell != "▌" && cell != "▐" && cell != "█" {
			quadrants++
		}
	}
	if quadrants == 0 {
		t.Fatalf("the Coarse Fish has no quadrants in it:\n%s", h.screen())
	}
	if a0, b0, a1, b1, _ := fishBounds(rows); a0 != x0 || b0 != y0 || a1 != x1 || b1 != y1 {
		t.Errorf("the Fish moved from (%d,%d)-(%d,%d) to (%d,%d)-(%d,%d) when its Detail changed:\n%s",
			x0, y0, x1, y1, a0, b0, a1, b1, h.screen())
	}
}

// coarse reports whether the Fish in rows is drawn in Coarse Detail: in
// quadrants, with no sextant anywhere in it.
func coarse(rows []string) bool {
	cs := fishCells(rows)
	for _, c := range cs {
		if isSextant(cells(rows[c[1]])[c[0]]) {
			return false
		}
	}
	return len(cs) > 0
}

func TestEachTerminalOpensInTheDetailItIsGuessedToShow(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		env      map[string]string
		terminal string // the name the Detail picked in it is remembered by
		coarse   bool
	}{
		{"Ghostty", map[string]string{"TERM_PROGRAM": "ghostty"}, "ghostty", false},
		{"WezTerm", map[string]string{"TERM_PROGRAM": "WezTerm"}, "WezTerm", false},
		{"Rio", map[string]string{"TERM_PROGRAM": "rio"}, "rio", false},
		{"Contour", map[string]string{"TERM_PROGRAM": "contour"}, "contour", false},
		{"kitty", map[string]string{"KITTY_WINDOW_ID": "1", "TERM": "xterm-kitty"}, "kitty", false},
		{"kitty over ssh", map[string]string{"TERM": "xterm-kitty"}, "kitty", false},
		{"Alacritty", map[string]string{"ALACRITTY_WINDOW_ID": "9", "TERM": "alacritty"}, "alacritty", false},
		{"foot", map[string]string{"TERM": "foot"}, "foot", false},
		{"foot-extra", map[string]string{"TERM": "foot-extra"}, "foot", false},

		{"iTerm2 3.6.7", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.6.7"}, "iTerm.app", false},
		{"iTerm2 3.7.0beta2", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.7.0beta2"}, "iTerm.app", false},
		{"iTerm2 3.6.6", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.6.6"}, "iTerm.app", true},
		{"iTerm2 3.5.14", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.5.14"}, "iTerm.app", true},
		{"iTerm2 without a version", map[string]string{"TERM_PROGRAM": "iTerm.app"}, "iTerm.app", true},
		{"iTerm2 with a garbled version", map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "x.y"}, "iTerm.app", true},
		{"VS Code 1.108", map[string]string{"TERM_PROGRAM": "vscode", "TERM_PROGRAM_VERSION": "1.108.0"}, "vscode", false},
		{"VS Code 1.107", map[string]string{"TERM_PROGRAM": "vscode", "TERM_PROGRAM_VERSION": "1.107.2"}, "vscode", true},
		{"VTE 0.60", map[string]string{"VTE_VERSION": "6000"}, "vte", false},
		{"VTE 0.58", map[string]string{"VTE_VERSION": "5802"}, "vte", true},
		{"Konsole 22.04", map[string]string{"KONSOLE_VERSION": "220400"}, "konsole", false},
		{"Konsole 21.12", map[string]string{"KONSOLE_VERSION": "211203"}, "konsole", true},

		{"Terminal.app", map[string]string{"TERM_PROGRAM": "Apple_Terminal"}, "Apple_Terminal", true},
		{"Warp", map[string]string{"TERM_PROGRAM": "WarpTerminal"}, "WarpTerminal", true},
		{"Windows Terminal", map[string]string{"WT_SESSION": "abc"}, "windows-terminal", true},
		{"tmux inside kitty", map[string]string{"TERM_PROGRAM": "tmux", "KITTY_WINDOW_ID": "1", "TMUX": "/tmp/t"}, "tmux", true},
		{"tmux without TERM_PROGRAM", map[string]string{"TMUX": "/tmp/t", "TERM": "tmux-256color", "KITTY_WINDOW_ID": "1"}, "tmux", true},
		{"herdr inside Ghostty", map[string]string{"TERM_PROGRAM": "herdr", "GHOSTTY_RESOURCES_DIR": "/x"}, "herdr", true},
		{"a plain xterm", map[string]string{"TERM": "xterm-256color"}, "xterm-256color", true},
		{"nothing at all", map[string]string{}, "unknown", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var env []string
			for k, v := range c.env {
				env = append(env, k, v)
			}
			h := newHarness(t, inTerminal(env...), named("Bubbles"))
			h.resize(80, 24)
			if got := coarse(h.view()); got != c.coarse {
				t.Errorf("the Fish opens Coarse: %v, want %v:\n%s", got, c.coarse, h.screen())
			}
			h.press("d")
			raw, err := os.ReadFile(filepath.Join(h.dataDir, "detail.json"))
			if err != nil {
				t.Fatalf("picking a Detail saved nothing: %v", err)
			}
			var picked map[string]string
			if err := json.Unmarshal(raw, &picked); err != nil {
				t.Fatalf("detail.json is not JSON: %v\n%s", err, raw)
			}
			want := "coarse"
			if c.coarse {
				want = "fine"
			}
			if len(picked) != 1 || picked[c.terminal] != want {
				t.Errorf("detail.json is %v, want {%q: %q}", picked, c.terminal, want)
			}
		})
	}
}

func TestTheDetailPickedIsRememberedForThatTerminalOnly(t *testing.T) {
	t.Parallel()
	terminalApp := inTerminal("TERM_PROGRAM", "Apple_Terminal")
	h := newHarness(t, terminalApp, named("Bubbles"))
	h.resize(80, 24)
	h.press("d") // Fine, though Terminal.app is guessed Coarse
	h.press("q")

	again := newHarness(t, terminalApp, withDataDir(h.dataDir))
	again.resize(80, 24)
	if coarse(again.view()) {
		t.Errorf("Terminal.app forgot the Fine Detail picked in it:\n%s", again.screen())
	}
	again.press("q")

	elsewhere := newHarness(t, inTerminal("TERM_PROGRAM", "WarpTerminal"), withDataDir(h.dataDir))
	elsewhere.resize(80, 24)
	if !coarse(elsewhere.view()) {
		t.Errorf("Warp took the Detail picked in Terminal.app, not its own guess:\n%s", elsewhere.screen())
	}
}

func TestTheDetailPickedOutlivesARelease(t *testing.T) {
	t.Parallel()
	h := newHarness(t, inTerminal("TERM_PROGRAM", "Apple_Terminal"), named("Bubbles"))
	h.resize(80, 24)
	h.press("d")
	h.press("r")
	h.press("y")
	h.advance(app.FarewellDuration)
	h.press("enter")
	h.nameFish("Pebble")
	h.press("q")

	again := newHarness(t, inTerminal("TERM_PROGRAM", "Apple_Terminal"), withDataDir(h.dataDir))
	again.resize(80, 24)
	if coarse(again.view()) {
		t.Errorf("the Fine Detail picked before a Release was forgotten:\n%s", again.screen())
	}
}

func TestABrokenDetailFileStillOpensTheTankInTheGuessedDetail(t *testing.T) {
	t.Parallel()
	for name, content := range map[string]string{
		"not JSON":        "{fine",
		"an unknown word": `{"Apple_Terminal": "tiny"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "detail.json"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			h := newHarness(t, inTerminal("TERM_PROGRAM", "Apple_Terminal"), withDataDir(dir), named("Bubbles"))
			h.resize(80, 24)
			if !coarse(h.view()) {
				t.Errorf("with a detail.json of %s, the Tank isn't open in the guessed Coarse:\n%s", name, h.screen())
			}
		})
	}
}

// boxesHint is what the status line says after the user picks Fine in a
// terminal that may not show it.
const boxesHint = "Boxes instead of a fish? Press d for Coarse"

func TestPickingFineWhereItMayNotShowSaysHowToGoBack(t *testing.T) {
	t.Parallel()
	h := newHarness(t, inTerminal("TERM_PROGRAM", "Apple_Terminal"), named("Bubbles"))
	h.resize(80, 24)
	h.press("d")
	if !strings.Contains(h.statusLine(), boxesHint) {
		t.Fatalf("after picking Fine in Terminal.app the status line is %q, want the hint %q", h.statusLine(), boxesHint)
	}
	h.advance(app.DetailHintDuration)
	if strings.Contains(h.statusLine(), boxesHint) {
		t.Errorf("the hint is still up %v later: %q", app.DetailHintDuration, h.statusLine())
	}

	h.press("d") // back to Coarse
	if strings.Contains(h.statusLine(), boxesHint) {
		t.Errorf("going back to Coarse shows the hint: %q", h.statusLine())
	}
	h.press("d") // Fine again
	h.press("1") // any other key takes it down
	if strings.Contains(h.statusLine(), boxesHint) {
		t.Errorf("the hint is still up after another key: %q", h.statusLine())
	}
}

func TestPickingFineWhereItAlwaysShowsSaysNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles")) // Ghostty
	h.resize(80, 24)
	h.press("d")
	h.press("d") // Fine again
	if coarse(h.view()) {
		t.Fatalf("d twice left the Fish Coarse:\n%s", h.screen())
	}
	if strings.Contains(h.statusLine(), boxesHint) {
		t.Errorf("picking Fine in Ghostty shows the hint: %q", h.statusLine())
	}
}

func TestDChangesTheDetailOnTheSpeciesPicker(t *testing.T) {
	t.Parallel()
	h := newHarness(t) // first launch: the picker is up
	h.resize(80, 24)
	if coarse(h.view()) {
		t.Fatalf("the picker's Fish starts Coarse in Ghostty:\n%s", h.screen())
	}
	h.press("d")
	if !coarse(h.view()) {
		t.Errorf("d on the picker left its Fish Fine:\n%s", h.screen())
	}
}

func TestWhileNamingDIsALetter(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.resize(80, 24)
	h.press("enter")
	h.nameFish("Eddy")
	if !strings.Contains(h.statusLine(), "Eddy") {
		t.Errorf("the Fish isn't called Eddy: %q", h.statusLine())
	}
	if coarse(h.view()) {
		t.Errorf("typing d into a name changed the Detail:\n%s", h.screen())
	}
}

func TestACoarseMonochromeFishStillHasAHoleForItsEye(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles"), monochrome())
	h.resize(80, 24)
	h.press("d")
	turned := map[bool]bool{}
	for i := range int(20 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		rows := h.view()
		s, ok := findFish(h.rawView())
		if !ok || !coarse(rows) {
			t.Fatalf("frame %d: no Coarse Fish found in monochrome:\n%s", i, h.screen())
		}
		if !eyeHole(rows, art, s.left, s.top, s.facingLeft, inCoarse) {
			t.Fatalf("frame %d: the Coarse Fish's eye is not a hole in its silhouette:\n%s", i, h.screen())
		}
		turned[s.facingLeft] = true
	}
	if len(turned) < 2 {
		t.Error("the Fish never turned in 20 seconds, so only one way was checked")
	}
}

func TestDChangesTheDetailWhileAskedToReleaseWithoutHidingTheQuestion(t *testing.T) {
	t.Parallel()
	h := newHarness(t, inTerminal("TERM_PROGRAM", "Apple_Terminal"), named("Bubbles"))
	h.resize(80, 24)
	h.press("r")
	h.press("d") // Fine, where it may show as boxes
	if coarse(h.view()) {
		t.Errorf("d while asked to release left the Fish Coarse:\n%s", h.screen())
	}
	if !strings.Contains(h.statusLine(), "Release Bubbles to the sea?") {
		t.Errorf("after d the release question is gone: %q", h.statusLine())
	}
}

func TestDChangesTheDetailDuringTheFarewell(t *testing.T) {
	t.Parallel()
	h := newHarness(t, named("Bubbles"))
	h.resize(80, 24)
	h.press("r")
	h.press("y")
	h.advance(app.FrameInterval)
	h.press("d")
	if !coarse(h.view()) {
		t.Errorf("d during the Farewell left the Fish Fine:\n%s", h.screen())
	}
}
