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
	"github.com/Kyu-seok/shellfish/internal/render"
	"github.com/Kyu-seok/shellfish/internal/store"
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
	// Monochrome draws without colour, for NO_COLOR or a terminal with too
	// few colours to show the Fish.
	Monochrome bool
}

// Model is the running app.
type Model struct {
	deps          Deps
	width, height int
	tank          *tank.Tank // nil until the window size is known
	naming        bool
	name          []rune // the name typed so far on the naming prompt
	nameErr       error  // why the last name was rejected
	fish          fish.Fish
	err           error     // a failure to open the Tank, shown instead of it
	saveErr       error     // why the last save failed, if it did
	now           time.Time // the time of the latest tick
	lastSaved     time.Time // when the Fish was last written to the data folder
	lock          *store.Lock
	refusal       string        // why this launch was refused, if another Tank is open
	greeting      string        // the welcome-back greeting, if the user was away a while
	greetUntil    time.Time     // when the greeting goes away by itself
	foodKind      fish.FoodKind // what a click or f drops
	quitting      bool          // the user has closed the Tank
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
// non-zero if the launch was refused because another Tank is open, or if the
// Fish could not be saved on quitting.
func (m Model) ExitStatus() int {
	if m.refusal != "" || m.quitting && m.saveErr != nil {
		return 1
	}
	return 0
}

// ExitMessage is what to print after the app quits, if anything.
func (m Model) ExitMessage() string {
	if m.quitting && m.saveErr != nil {
		return "shellfish: " + m.saveErr.Error()
	}
	return m.refusal
}

// Close releases the data folder's lock. Quitting releases it too; Close is
// safe to call again afterwards.
func (m Model) Close() error { return m.lock.Release() }

// quit closes the Tank: it saves the Fish, if there is one, then releases the
// lock so another launch can open it. Every quit key goes through here.
func (m Model) quit() (tea.Model, tea.Cmd) {
	if m.hasFish() {
		m = m.save(m.deps.Clock.Now())
	}
	m.quitting = true
	m.Close()
	return m, tea.Quit
}

// Update handles a terminal message (window resizes and keys) or a tick.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		w, h := tankArea(msg.Width, msg.Height)
		if m.tank == nil {
			m.tank = tank.New(m.deps.Rand, w, h)
		} else {
			m.tank.Resize(w, h)
		}
	case TickMsg:
		m.now = msg.Time
		if m.hasFish() {
			m.fish = m.fish.CatchUp(m.now)
		}
		if m.tank != nil {
			if meals := m.tank.Step(FrameInterval, m.fish.Mood(), !m.hasFish() || m.fish.Full()); len(meals) > 0 {
				for _, meal := range meals {
					m.fish = m.fish.Eat(meal.Kind)
				}
				m = m.save(m.now)
			}
		}
		// A clock that moved backwards saves now instead of waiting to catch up.
		if since := m.now.Sub(m.lastSaved); m.hasFish() && (since >= SaveInterval || since < 0) {
			m = m.save(m.now)
		}
		return m, m.deps.Clock.Tick(FrameInterval)
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && m.tankShown() {
			m.tank.Drop(m.foodKind, msg.X, msg.Y)
		}
	case tea.KeyMsg:
		if m.refusal != "" {
			return m, tea.Quit
		}
		if m.naming {
			return m.updateNaming(msg)
		}
		switch key := msg.String(); key {
		case "q", "esc", "ctrl+c":
			return m.quit()
		case "f", " ":
			if m.tankShown() {
				m.tank.DropNearTop(m.foodKind)
			}
		case "1", "2", "3":
			m.foodKind = fish.FoodKinds[key[0]-'1']
		}
	}
	return m, nil
}

// statusRows is how many rows below the Tank the status line takes.
const statusRows = 1

// tankArea is the part of a width×height window that the Tank fills. The
// Tank is sized only through here, so rows the window keeps for anything else
// are subtracted in this one place.
func tankArea(width, height int) (w, h int) {
	return width, max(height-statusRows, 0)
}

// faces is the Fish's face for each Mood.
var faces = map[fish.Mood]string{
	fish.Content:  "◕ ᴗ",
	fish.Hungry:   "• _",
	fish.Starving: "; ﹏",
}

// tankShown reports whether the Tank is open and drawn, so Food can be
// dropped into it.
func (m Model) tankShown() bool {
	return m.hasFish() && m.tank != nil && m.width >= MinTankWidth && m.height >= MinTankHeight
}

// View draws the Tank to fill the window, with the status line below it.
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
	snap := m.tank.Snapshot()
	look := render.Look{Face: faces[m.fish.Mood()]}
	if !m.deps.Monochrome {
		look.Colour = string(m.fish.Colour)
	}
	water := render.Tank(snap, look)
	if len(water) > 0 {
		m.drawGreeting(water[:len(water)-1]) // over the water, not the sand
	}
	rows := append(water, m.statusLine())
	return strings.Join(rows, "\n")
}

// keyHint teaches the controls at the right end of the status line.
const keyHint = "click/f feed · 1-3 food · q quit"

// saveFailed takes the key hint's place while the Fish can't be saved.
const saveFailed = "⚠ couldn't save"

// hintStyle is faint, which is not a colour, so it is kept in monochrome.
var hintStyle = lipgloss.NewStyle().Faint(true)

// statusLine is exactly the window's width: the Fish's facts on the left and
// a faint key hint at the right end, which makes way when there is no room.
// The selected Food Kind, last of the facts, shrinks from its glyph and name
// to its glyph alone before the hint makes way for it.
func (m Model) statusLine() string {
	glyph := render.FoodGlyph(m.foodKind)
	named := glyph + " " + m.foodKind.String()
	hint := hintStyle.Render(keyHint)
	if m.saveErr != nil {
		hint = saveFailed
	}
	for _, food := range []string{named, glyph} {
		facts := m.statusFacts(food)
		if gap := m.width - ansi.StringWidth(facts) - ansi.StringWidth(hint) - 1; gap >= 2 {
			return facts + strings.Repeat(" ", gap) + hint + " "
		}
	}
	return fit(m.statusFacts(named, glyph), m.width)
}

// statusFacts shows the Fish's name, then its Hunger meter, its Age and the
// first of foods (ways of showing the selected Food Kind) that fits, each
// only if it fits whole in the window's width, so it never ends in a
// dangling "·".
func (m Model) statusFacts(foods ...string) string {
	days := m.fish.Age(m.now)
	unit := "days"
	if days == 1 {
		unit = "day"
	}
	facts := " " + m.fish.Name
	for _, item := range [][]string{
		{"Hunger " + hungerMeter(m.fish.Hunger)},
		{fmt.Sprintf("Age %d %s", days, unit)},
		foods,
	} {
		for _, way := range item {
			if next := facts + " · " + way; ansi.StringWidth(next) <= m.width {
				facts = next
				break
			}
		}
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
// and writes it to the data folder. A failed save leaves the Tank open, with
// the failure on the status line, and the next periodic save tries again.
func (m Model) save(now time.Time) Model {
	m.fish = m.fish.CatchUp(now)
	m.fish.LastSeen = now
	m.lastSaved = now
	m.saveErr = store.Save(m.deps.DataDir, m.fish)
	return m
}
