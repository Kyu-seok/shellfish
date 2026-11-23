// Package app is the Bubble Tea model for shellfish. It is built only from the
// injected Deps, which is the single seam tests drive it through.
package app

import (
	"errors"
	"math/rand/v2"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"

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
	picking       bool       // the Species picker is up
	pick          int        // which of fish.AllSpecies the picker shows
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
	away          time.Duration // how long the user was away, if long enough to greet them
	greetUntil    time.Time     // when the greeting goes away by itself
	foodKind      fish.FoodKind // what a click or f drops
	quitting      bool          // the user has closed the Tank
}

// New builds the app from its dependencies. It takes the data folder's lock,
// then loads the saved Fish, or starts the first-launch flow (Species picker,
// then naming) if there is none. A Fish saved before Species existed is shown
// the Species picker before its Tank opens. If
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
		m.picking = true
	case err != nil:
		m.err = err
	default:
		m.away = awayFor(f, m.now)
		m.fish = f.CatchUp(m.now)
		if f.Species == fish.NoSpecies {
			m.picking = true // the greeting waits until the Tank opens
		} else {
			m.greetUntil = m.now.Add(GreetingDuration)
		}
	}
	return m
}

// species is the Species drawn in the Tank: the one the picker shows while
// it is up, and the Fish's own after.
func (m Model) species() fish.Species {
	if m.picking {
		return fish.AllSpecies[m.pick]
	}
	return m.fish.Species
}

// awayFor is how long the user was away from f at now, if it was more than
// GreetAfter, so the Tank greets them; otherwise zero.
func awayFor(f fish.Fish, now time.Time) time.Duration {
	if away := now.Sub(f.LastSeen); !f.LastSeen.IsZero() && away > GreetAfter {
		return away
	}
	return 0
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
			m.tank = tank.New(m.deps.Rand, w, h, tank.BodyOf(m.species()))
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
		if m.picking {
			return m.updatePicking(msg)
		}
		if m.naming {
			return m.updateNaming(msg)
		}
		// Keys typed in quick succession arrive as one message: act on each.
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 1 {
			var model tea.Model = m
			var cmd tea.Cmd
			for _, r := range msg.Runes {
				model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
				if cmd != nil {
					return model, cmd
				}
			}
			return model, nil
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

// tankArea is the part of a width×height window that the Tank fills. The
// Tank is sized only through here, so rows the window keeps for anything else
// are subtracted in this one place.
func tankArea(width, height int) (w, h int) {
	return width, max(height-render.StatusRows, 0)
}

// tankShown reports whether the Tank is open and drawn, so Food can be
// dropped into it.
func (m Model) tankShown() bool {
	return m.hasFish() && m.tank != nil && m.width >= MinTankWidth && m.height >= MinTankHeight
}

// View draws the Tank to fill the window, with the status line below it,
// or whatever takes its place.
func (m Model) View() string {
	switch {
	case m.width <= 0 || m.height <= 0:
		return ""
	case m.refusal != "":
		return render.Message(m.width, m.height, m.refusal)
	case m.err != nil:
		return render.Failure(m.width, m.height, m.err)
	case m.naming:
		return render.Naming(m.width, m.height, string(m.name), m.nameErr)
	case m.width < MinTankWidth || m.height < MinTankHeight:
		return render.TooSmall(m.width, m.height)
	}
	sc := render.Scene{
		Tank: m.tank.Snapshot(),
		Look: render.Look{Art: m.species().Art(), Mood: m.fish.Mood(), Monochrome: m.deps.Monochrome},
		Status: render.Status{
			Name:       m.fish.Name,
			Hunger:     m.fish.Hunger,
			Age:        m.fish.Age(m.now),
			Food:       m.foodKind,
			SaveFailed: m.saveErr != nil,
		},
	}
	if m.picking {
		sc.Picking = m.species()
	}
	if m.now.Before(m.greetUntil) {
		sc.Away = m.away
	}
	return render.Screen(sc)
}

// updatePicking handles a key on the Species picker: ←/→ browse, Enter
// chooses, and Esc and Ctrl-C quit without saving. Other keys, q included,
// do nothing.
func (m Model) updatePicking(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(fish.AllSpecies)
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return m.quit()
	case tea.KeyLeft:
		m.pick = (m.pick + n - 1) % n
	case tea.KeyRight:
		m.pick = (m.pick + 1) % n
	case tea.KeyEnter:
		m.fish.Species, m.picking = m.species(), false
		if m.fish.Name == "" {
			m.naming = true // a new Fish is named next
			return m, nil
		}
		// A Fish saved before Species existed opens its Tank now.
		m.now = m.deps.Clock.Now()
		m.greetUntil = m.now.Add(GreetingDuration)
		m = m.save(m.now)
		return m, nil
	default:
		return m, nil
	}
	if m.tank != nil {
		m.tank.Reshape(tank.BodyOf(m.species()))
	}
	return m, nil
}

// updateNaming handles a key on the naming prompt. Letters (q included) go
// into the name, so only Esc and Ctrl-C quit here.
func (m Model) updateNaming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return m.quit()
	case tea.KeyEnter:
		f, err := fish.Hatch(string(m.name), m.fish.Species, m.deps.Clock.Now())
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

// hasFish reports whether a Fish is living in this Tank: it has a Species, has been named and
// loaded without error, and this launch holds the lock. A refused launch has
// no Fish of its own, so it never saves.
func (m Model) hasFish() bool { return m.refusal == "" && !m.picking && !m.naming && m.err == nil }

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
