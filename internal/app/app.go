// Package app is the Bubble Tea model for shellfish. It is built only from the
// injected Deps, which is the single seam tests drive it through.
package app

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Kyu-seok/shellfish/internal/render"
	"github.com/Kyu-seok/shellfish/internal/tank"
)

// Clock supplies the current time and the animation tick source.
type Clock interface {
	Now() time.Time
	// Tick returns a command that delivers one TickMsg after d.
	Tick(d time.Duration) tea.Cmd
}

// TickMsg is one animation frame, delivered by the Clock.
type TickMsg struct{ Time time.Time }

// Deps are everything the app needs from the outside world.
type Deps struct {
	Clock   Clock
	Rand    *rand.Rand
	DataDir string
}

// Model is the running app.
type Model struct {
	deps Deps
	tank *tank.Tank // nil until the window size is known
}

// New builds the app from its dependencies.
func New(deps Deps) Model {
	return Model{deps: deps}
}

// Init starts the animation ticks.
func (m Model) Init() tea.Cmd { return m.deps.Clock.Tick(FrameInterval) }

// Update handles a terminal message (window resizes and keys) or a tick.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := tankArea(msg.Width, msg.Height)
		if m.tank == nil {
			m.tank = tank.New(m.deps.Rand, w, h)
		} else {
			m.tank.Resize(w, h)
		}
	case TickMsg:
		if m.tank != nil {
			m.tank.Step(FrameInterval)
		}
		return m, m.deps.Clock.Tick(FrameInterval)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// tankArea is the part of a width×height window that the Tank fills. The
// Tank is sized only through here, so rows the window keeps for anything else
// are subtracted in this one place.
func tankArea(width, height int) (w, h int) {
	return width, height
}

// View draws the Tank to fill the window.
func (m Model) View() string {
	if m.tank == nil {
		return ""
	}
	return strings.Join(render.Tank(m.tank.Snapshot()), "\n")
}
