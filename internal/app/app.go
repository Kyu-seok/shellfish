// Package app is the Bubble Tea model for shellfish. It is built only from the
// injected Deps, which is the single seam tests drive it through.
package app

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// Clock supplies the current time.
type Clock interface {
	Now() time.Time
}

// Deps are everything the app needs from the outside world.
type Deps struct {
	Clock   Clock
	Rand    *rand.Rand
	DataDir string
}

// Model is the running app.
type Model struct {
	deps          Deps
	width, height int
}

// New builds the app from its dependencies.
func New(deps Deps) Model {
	return Model{deps: deps}
}

// Init starts the app; nothing runs until the first message arrives.
func (m Model) Init() tea.Cmd { return nil }

// Update handles a terminal message: window resizes and keys.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// fishSprite is the puffball facing right.
var fishSprite = []string{
	"   .-~~-.",
	"><(  ◕ ᴗ )",
	"   `-..-'",
}

const sand = "░"

// View draws the Tank to fill the window, with the Fish in the middle above a sandy floor.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	rows := make([]string, m.height)
	floorRow := m.height - 1

	spriteW := 0
	for _, line := range fishSprite {
		spriteW = max(spriteW, ansi.StringWidth(line))
	}
	top := (floorRow - len(fishSprite)) / 2
	left := max((m.width-spriteW)/2, 0)

	for y := 0; y < floorRow; y++ {
		row := ""
		if i := y - top; i >= 0 && i < len(fishSprite) {
			row = strings.Repeat(" ", left) + fishSprite[i]
		}
		rows[y] = fit(row, m.width)
	}
	rows[floorRow] = strings.Repeat(sand, m.width)
	return strings.Join(rows, "\n")
}

// fit pads or truncates s to exactly w cells.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	return s + strings.Repeat(" ", w-ansi.StringWidth(s))
}
