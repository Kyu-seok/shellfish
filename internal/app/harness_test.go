package app_test

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/Kyu-seok/shellfish/internal/app"
)

// start is the fake clock's time when a harness is built, unless overridden.
var start = time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

// fakeClock is moved by hand. Its Tick schedules nothing: the harness
// delivers ticks itself in advance.
type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time             { return c.now }
func (c *fakeClock) Tick(time.Duration) tea.Cmd { return nil }

// harness drives the app exactly as a terminal would: by sending it messages
// and reading the Tank it renders. Build several harnesses on the same
// dataDir to simulate quitting and launching again.
type harness struct {
	t       *testing.T
	clock   *fakeClock
	dataDir string
	model   tea.Model
	last    tea.Cmd

	afterLaunch []func()
}

type option func(*harness)

// withDataDir launches on an existing data folder, as a later launch would.
func withDataDir(dir string) option { return func(h *harness) { h.dataDir = dir } }

// named launches and names the Fish on its first-launch prompt, so the Tank
// is open when the test starts.
func named(name string) option {
	return func(h *harness) { h.afterLaunch = append(h.afterLaunch, func() { h.nameFish(name) }) }
}

// at launches with the fake clock at t.
func at(t time.Time) option { return func(h *harness) { h.clock.now = t } }

func newHarness(t *testing.T, opts ...option) *harness {
	t.Helper()
	lipgloss.SetColorProfile(termenv.TrueColor)
	h := &harness{t: t, clock: &fakeClock{now: start}}
	for _, o := range opts {
		o(h)
	}
	if h.dataDir == "" {
		h.dataDir = t.TempDir()
	}
	m := app.New(app.Deps{
		Clock:   h.clock,
		Rand:    rand.New(rand.NewPCG(1, 2)),
		DataDir: h.dataDir,
	})
	h.model, h.last = m, m.Init()
	for _, f := range h.afterLaunch {
		f()
	}
	return h
}

func (h *harness) send(msgs ...tea.Msg) {
	for _, msg := range msgs {
		h.model, h.last = h.model.Update(msg)
	}
}

func (h *harness) resize(width, height int) {
	h.send(tea.WindowSizeMsg{Width: width, Height: height})
}

// press sends one key, named as tea.KeyMsg.String() names it: "q", "esc",
// "ctrl+c", "enter", "backspace", " ", or any single printable character.
func (h *harness) press(key string) {
	switch key {
	case "esc":
		h.send(tea.KeyMsg{Type: tea.KeyEsc})
	case "ctrl+c":
		h.send(tea.KeyMsg{Type: tea.KeyCtrlC})
	case "enter":
		h.send(tea.KeyMsg{Type: tea.KeyEnter})
	case "backspace":
		h.send(tea.KeyMsg{Type: tea.KeyBackspace})
	case " ":
		h.send(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})
	default:
		h.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	}
}

// typeText types each character of s.
func (h *harness) typeText(s string) {
	for _, r := range s {
		h.press(string(r))
	}
}

// nameFish answers the first-launch prompt.
func (h *harness) nameFish(name string) {
	h.typeText(name)
	h.press("enter")
}

// advance moves fake time forward by d, delivering one animation tick per
// frame, as the real clock would while the Tank is open.
func (h *harness) advance(d time.Duration) {
	for d > 0 {
		step := min(app.FrameInterval, d)
		h.clock.now = h.clock.now.Add(step)
		d -= step
		h.send(app.TickMsg{Time: h.clock.now})
	}
}

// view is the rendered Tank as plain text, one string per terminal row.
func (h *harness) view() []string {
	return strings.Split(ansi.Strip(h.model.View()), "\n")
}

// rawView is the rendered Tank with its colour escape codes, one string per
// terminal row. The harness renders as a true-colour terminal would.
func (h *harness) rawView() []string {
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
