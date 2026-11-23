package app_test

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/Kyu-seok/shellfish/internal/app"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

// harness drives the app exactly as a terminal would: by sending it messages
// and reading the Tank it renders.
type harness struct {
	t     *testing.T
	model tea.Model
	last  tea.Cmd
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	m := app.New(app.Deps{
		Clock:   &fakeClock{now: time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)},
		Rand:    rand.New(rand.NewPCG(1, 2)),
		DataDir: t.TempDir(),
	})
	return &harness{t: t, model: m, last: m.Init()}
}

func (h *harness) send(msgs ...tea.Msg) {
	for _, msg := range msgs {
		h.model, h.last = h.model.Update(msg)
	}
}

func (h *harness) resize(width, height int) {
	h.send(tea.WindowSizeMsg{Width: width, Height: height})
}

func (h *harness) view() []string {
	return strings.Split(h.model.View(), "\n")
}

// quitRequested reports whether the last message made the app ask to quit.
func (h *harness) quitRequested() bool {
	if h.last == nil {
		return false
	}
	_, ok := h.last().(tea.QuitMsg)
	return ok
}

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
	for name, key := range map[string]tea.KeyMsg{
		"q":      {Type: tea.KeyRunes, Runes: []rune{'q'}},
		"esc":    {Type: tea.KeyEsc},
		"ctrl+c": {Type: tea.KeyCtrlC},
	} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t)
			h.resize(80, 24)
			h.send(key)
			if !h.quitRequested() {
				t.Errorf("pressing %s did not quit", name)
			}
		})
	}
}

func TestOtherKeysDoNotQuit(t *testing.T) {
	h := newHarness(t)
	h.resize(80, 24)
	h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if h.quitRequested() {
		t.Error("pressing x quit the program")
	}
}
