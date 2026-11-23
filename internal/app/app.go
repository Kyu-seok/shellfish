// Package app is the Bubble Tea model for shellfish. It is built only from the
// injected Deps, which is the single seam tests drive it through.
package app

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Kyu-seok/shellfish/internal/fish"
	"github.com/Kyu-seok/shellfish/internal/store"
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
	deps          Deps
	width, height int
	naming        bool
	name          []rune // the name typed so far on the naming prompt
	nameErr       error  // why the last name was rejected
	fish          fish.Fish
	err           error     // a store failure, shown instead of the Tank
	now           time.Time // the time of the latest tick
	lock          *store.Lock
	refusal       string // why this launch was refused, if another Tank is open
}

// New builds the app from its dependencies. It takes the data folder's lock,
// then loads the saved Fish, or starts the naming flow if there is none. If
// another Tank holds the lock, the launch is refused: see ExitStatus.
func New(deps Deps) Model {
	m := Model{deps: deps, now: deps.Clock.Now()}
	lock, err := store.Acquire(deps.DataDir)
	if err != nil {
		if errors.Is(err, store.ErrLocked) {
			m.refusal = refusal(deps.DataDir)
		} else {
			m.err = err
		}
		return m
	}
	m.lock = lock
	f, err := store.Load(deps.DataDir)
	switch {
	case errors.Is(err, store.ErrNoFish):
		m.naming = true
	case err != nil:
		m.err = err
	default:
		m.fish = f
	}
	return m
}

// refusal is the message for a launch while another Tank is open.
func refusal(dataDir string) string {
	name := "Your Fish"
	if f, err := store.Load(dataDir); err == nil {
		name = f.Name
	}
	return name + " is already swimming in another terminal"
}

// Init starts the animation ticks, or quits at once if the launch was refused.
func (m Model) Init() tea.Cmd {
	if m.refusal != "" {
		return tea.Quit
	}
	return m.deps.Clock.Tick(FrameInterval)
}

// ExitStatus is the status the process should exit with once the app quits:
// non-zero if the launch was refused because another Tank is open.
func (m Model) ExitStatus() int {
	if m.refusal != "" {
		return 1
	}
	return 0
}

// ExitMessage is what to print after the app quits, if anything.
func (m Model) ExitMessage() string { return m.refusal }

// Close releases the data folder's lock. Quitting releases it too; Close is
// safe to call again afterwards.
func (m Model) Close() error { return m.lock.Release() }

// quit closes the Tank, releasing the lock so another launch can open it.
func (m Model) quit() (tea.Model, tea.Cmd) {
	m.Close()
	return m, tea.Quit
}

// Update handles a terminal message (window resizes and keys) or a tick.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case TickMsg:
		m.now = msg.Time
		return m, m.deps.Clock.Tick(FrameInterval)
	case tea.KeyMsg:
		if m.refusal != "" {
			return m, tea.Quit
		}
		if m.naming {
			return m.updateNaming(msg)
		}
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m.quit()
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
	if m.refusal != "" {
		return m.messageView(m.refusal)
	}
	if m.err != nil {
		return m.messageView("shellfish: "+m.err.Error(), "", "Press q to quit.")
	}
	if m.naming {
		return m.namingView()
	}
	rows := make([]string, m.height)
	statusRow := m.height - 1
	floorRow := statusRow - 1

	spriteW := 0
	for _, line := range fishSprite {
		spriteW = max(spriteW, ansi.StringWidth(line))
	}
	body := lipgloss.NewStyle().Foreground(lipgloss.Color(m.fish.Colour))
	top := (floorRow - len(fishSprite)) / 2
	left := max((m.width-spriteW)/2, 0)

	for y := 0; y < floorRow; y++ {
		row := ""
		if i := y - top; i >= 0 && i < len(fishSprite) {
			row = strings.Repeat(" ", left) + body.Render(fishSprite[i])
		}
		rows[y] = fit(row, m.width)
	}
	rows[floorRow] = strings.Repeat(sand, m.width)
	rows[statusRow] = fit(m.statusLine(), m.width)
	return strings.Join(rows, "\n")
}

// statusLine shows the Fish's name and Age.
func (m Model) statusLine() string {
	days := m.fish.Age(m.now)
	unit := "days"
	if days == 1 {
		unit = "day"
	}
	return fmt.Sprintf(" %s · Age %d %s", m.fish.Name, days, unit)
}

// fit pads or truncates s to exactly w cells.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	return s + strings.Repeat(" ", w-ansi.StringWidth(s))
}

// namingView is the first-launch prompt asking what the Fish is called.
func (m Model) namingView() string {
	lines := []string{"What is your Fish called?", "", "> " + string(m.name)}
	if m.nameErr != nil {
		lines = append(lines, "", "Your Fish needs a name.")
	}
	return m.messageView(lines...)
}

// messageView centres a block of lines in the window, sharing one left edge.
func (m Model) messageView(lines ...string) string {
	rows := make([]string, m.height)
	top := max((m.height-len(lines))/2, 0)
	blockW := 0
	for _, l := range lines {
		blockW = max(blockW, ansi.StringWidth(l))
	}
	left := strings.Repeat(" ", max((m.width-blockW)/2, 0))
	for y := range rows {
		row := ""
		if i := y - top; i >= 0 && i < len(lines) {
			row = left + lines[i]
		}
		rows[y] = fit(row, m.width)
	}
	return strings.Join(rows, "\n")
}

// updateNaming handles a key on the naming prompt. Letters (q included) go
// into the name, so only Esc and Ctrl-C quit here.
func (m Model) updateNaming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return m.quit()
	case tea.KeyEnter:
		f, err := fish.Hatch(string(m.name), m.deps.Clock.Now(), m.deps.Rand)
		if err != nil {
			m.name, m.nameErr = nil, err
			return m, nil
		}
		m.fish, m.naming = f, false
		m.err = store.Save(m.deps.DataDir, f)
	case tea.KeyBackspace:
		if len(m.name) > 0 {
			m.name = m.name[:len(m.name)-1]
		}
	case tea.KeyRunes, tea.KeySpace:
		m.name, m.nameErr = append(slices.Clone(m.name), msg.Runes...), nil
	}
	return m, nil
}
