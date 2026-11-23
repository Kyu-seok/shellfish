// Package app is the Bubble Tea model for shellfish. It is built only from the
// injected Deps, which is the single seam tests drive it through.
package app

import (
	"errors"
	"fmt"
	"math"
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
	// Monochrome draws without colour, for NO_COLOR or a terminal with too
	// few colours to show the Fish.
	Monochrome bool
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
	lastSaved     time.Time // when the Fish was last written to the data folder
	lock          *store.Lock
	refusal       string    // why this launch was refused, if another Tank is open
	greeting      string    // the welcome-back greeting, if the user was away a while
	greetUntil    time.Time // when the greeting goes away by itself
}

// New builds the app from its dependencies. It takes the data folder's lock,
// then loads the saved Fish, or starts the naming flow if there is none. If
// another Tank holds the lock, the launch is refused: see ExitStatus.
func New(deps Deps) Model {
	now := deps.Clock.Now()
	m := Model{deps: deps, now: now, lastSaved: now}
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
		m.greeting = greeting(f, m.now)
		m.greetUntil = m.now.Add(GreetingDuration)
		m.fish = f.CatchUp(m.now)
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

// quit closes the Tank: it saves the Fish, if there is one, then releases the
// lock so another launch can open it. Every quit key goes through here.
func (m Model) quit() (tea.Model, tea.Cmd) {
	if m.hasFish() {
		m = m.save(m.deps.Clock.Now())
	}
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
		if m.hasFish() {
			m.fish = m.fish.CatchUp(m.now)
			// A clock that moved backwards saves now instead of waiting to catch up.
			if since := m.now.Sub(m.lastSaved); since >= SaveInterval || since < 0 {
				m = m.save(m.now)
			}
		}
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

// fishSprite is the puffball facing right, wearing face.
func fishSprite(face string) []string {
	return []string{
		"   .-~~-.",
		"><(  " + face + " )",
		"   `-..-'",
	}
}

// faces is the Fish's face for each Mood.
var faces = map[fish.Mood]string{
	fish.Content:  "◕ ᴗ",
	fish.Hungry:   "• _",
	fish.Starving: "; ﹏",
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
	if m.width < MinTankWidth || m.height < MinTankHeight {
		msg := ansi.Wordwrap("Make me a bigger tank 🐟", m.width, "")
		return m.messageView(strings.Split(msg, "\n")...)
	}
	rows := make([]string, m.height)
	statusRow := m.height - 1
	floorRow := statusRow - 1

	sprite := fishSprite(faces[m.fish.Mood()])
	spriteW := 0
	for _, line := range sprite {
		spriteW = max(spriteW, ansi.StringWidth(line))
	}
	body := lipgloss.NewStyle()
	if !m.deps.Monochrome {
		body = body.Foreground(lipgloss.Color(m.fish.Colour))
	}
	top := (floorRow - len(sprite)) / 2
	left := max((m.width-spriteW)/2, 0)

	for y := 0; y < floorRow; y++ {
		row := ""
		if i := y - top; i >= 0 && i < len(sprite) {
			row = strings.Repeat(" ", left) + body.Render(sprite[i])
		}
		rows[y] = fit(row, m.width)
	}
	m.drawGreeting(rows[:floorRow])
	rows[floorRow] = strings.Repeat(sand, m.width)
	rows[statusRow] = m.statusLine()
	return strings.Join(rows, "\n")
}

// keyHint teaches the controls at the right end of the status line.
const keyHint = "click/f feed · 1-3 food · q quit"

// hintStyle is faint, which is not a colour, so it is kept in monochrome.
var hintStyle = lipgloss.NewStyle().Faint(true)

// statusLine is exactly the window's width: the Fish's facts on the left and
// a faint key hint at the right end, which makes way when there is no room.
func (m Model) statusLine() string {
	facts := m.statusFacts()
	gap := m.width - ansi.StringWidth(facts) - ansi.StringWidth(keyHint) - 1
	if gap < 2 {
		return fit(facts, m.width)
	}
	return facts + strings.Repeat(" ", gap) + hintStyle.Render(keyHint) + " "
}

// statusFacts shows the Fish's name, its Hunger meter and its Age, as many of
// them as fit whole in the window's width, so it never ends in a dangling "·".
func (m Model) statusFacts() string {
	days := m.fish.Age(m.now)
	unit := "days"
	if days == 1 {
		unit = "day"
	}
	facts := " " + m.fish.Name
	for _, item := range []string{
		"Hunger " + hungerMeter(m.fish.Hunger),
		fmt.Sprintf("Age %d %s", days, unit),
	} {
		next := facts + " · " + item
		if ansi.StringWidth(next) > m.width {
			break
		}
		facts = next
	}
	return facts
}

// hungerMeter draws Hunger as HungerMeterCells cells, filled from the left
// as the Fish gets hungrier.
func hungerMeter(hunger float64) string {
	filled := int(math.Round(hunger / fish.MaxHunger * HungerMeterCells))
	filled = min(max(filled, 0), HungerMeterCells)
	return strings.Repeat("▰", filled) + strings.Repeat("▱", HungerMeterCells-filled)
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
		m = m.save(f.Born)
	case tea.KeyBackspace:
		if len(m.name) > 0 {
			m.name = m.name[:len(m.name)-1]
		}
	case tea.KeyRunes, tea.KeySpace:
		m.name, m.nameErr = append(slices.Clone(m.name), msg.Runes...), nil
	}
	return m, nil
}

// hasFish reports whether a Fish is living in this Tank: it has been named and
// loaded without error, and this launch holds the lock. A refused launch has
// no Fish of its own, so it never saves.
func (m Model) hasFish() bool { return m.refusal == "" && !m.naming && m.err == nil }

// save brings the Fish up to date at now, marks the Tank as last seen then,
// and writes it to the data folder.
func (m Model) save(now time.Time) Model {
	m.fish = m.fish.CatchUp(now)
	m.fish.LastSeen = now
	m.lastSaved = now
	m.err = store.Save(m.deps.DataDir, m.fish)
	return m
}
