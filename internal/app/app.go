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

// phase is what the app is doing, which decides what each key means and what
// is drawn.
type phase int

const (
	// inTank: the Fish is living in its Tank.
	inTank phase = iota
	// confirming: the Fish is living in its Tank, and the user is asked
	// whether to release it.
	confirming
	// farewell: the released Fish is swimming out of its Tank, saying bye.
	farewell
	// picking: the Species picker is up, choosing a Species for a new Fish.
	picking
	// naming: the naming prompt is up for a new Fish of the chosen Species.
	naming
	// pickingForSaved: the Species picker is up, choosing a Species for the
	// saved Fish, which was saved before Species existed.
	pickingForSaved
)

// live reports whether the saved Fish is living in its open Tank in p: it
// ages, eats and is saved. It lives on while the user is asked whether to
// release it and while it swims away.
func (p phase) live() bool { return p == inTank || p == confirming || p == farewell }

// picks reports whether the Species picker is up in p.
func (p phase) picks() bool { return p == picking || p == pickingForSaved }

// showsSaved reports whether the saved Fish is drawn in the Tank in p, rather
// than a new Fish yet to be hatched.
func (p phase) showsSaved() bool { return p.live() || p == pickingForSaved }

// Model is the running app.
type Model struct {
	deps          Deps
	width, height int
	tank          *tank.Tank // nil until the window size is known
	phase         phase
	// The picker and naming prompt keep their own state, apart from fish, so
	// that fish stays the saved Fish until a new one is named.
	pick          int          // which of fish.AllSpecies the picker shows
	chosenSpecies fish.Species // the Species chosen on the picker for the Fish being named
	name          []rune       // the name typed so far on the naming prompt
	nameErr       error        // why the last name was rejected
	// fish is the saved Fish: zero before the first one is named, and without
	// a Species if it was saved before Species existed.
	fish       fish.Fish
	err        error     // a failure to open the Tank, shown instead of it
	saveErr    error     // why the last save failed, if it did
	now        time.Time // the time of the latest tick
	lastSaved  time.Time // when the Fish was last written to the data folder
	lock       *store.Lock
	refusal    string        // why this launch was refused, if another Tank is open
	away       time.Duration // how long the user was away, if long enough to greet them
	greetUntil time.Time     // when the greeting goes away by itself
	byeUntil   time.Time     // when the farewell ends and the picker opens
	foodKind   fish.FoodKind // what a click on the water or f drops
	quitting   bool          // the user has closed the Tank
}

// New builds the app from its dependencies. It takes the data folder's lock,
// then loads the saved Fish, or starts the first-launch flow (Species picker,
// then naming) if there is none. A Fish saved before Species existed is shown
// the Species picker before its Tank opens. If another Tank holds the lock,
// the launch is refused: see ExitStatus.
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
		m = m.startPicking(picking)
	case err != nil:
		m.err = err
	default:
		m.away = awayFor(f, m.now)
		m.fish = f.CatchUp(m.now)
		if f.Species == fish.NoSpecies {
			m = m.startPicking(pickingForSaved) // the greeting waits until the Tank opens
		} else {
			m.greetUntil = m.now.Add(GreetingDuration)
		}
	}
	return m
}

// species is the Species drawn in the Tank: the one the picker shows while
// it is up, the chosen one while naming, and the Fish's own in the Tank.
func (m Model) species() fish.Species {
	switch {
	case m.phase.picks():
		return fish.AllSpecies[m.pick]
	case m.phase == naming:
		return m.chosenSpecies
	}
	return m.fish.Species
}

// startPicking opens the Species picker on the first Species, in phase p:
// picking for a new Fish, or pickingForSaved.
func (m Model) startPicking(p phase) Model {
	m.phase, m.pick = p, 0
	if m.tank != nil {
		m.tank.Reshape(tank.BodyOf(m.species()))
	}
	return m
}

// shown is the Fish drawn in the Tank: the saved Fish while it lives there,
// or is having its Species chosen, and otherwise a Fish yet to be hatched,
// so the picker and naming prompt for a new Fish show nothing of an old one.
func (m Model) shown() fish.Fish {
	if m.phase.showsSaved() {
		return m.fish
	}
	return fish.Fish{}
}

// startNaming opens an empty naming prompt for a new Fish of Species s.
func (m Model) startNaming(s fish.Species) Model {
	m.phase, m.chosenSpecies, m.name, m.nameErr = naming, s, nil, nil
	return m
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
			if meals := m.tank.Step(FrameInterval, m.shown().Mood(), !m.hasFish() || m.fish.Full()); len(meals) > 0 {
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
		if m.phase == farewell && !m.now.Before(m.byeUntil) {
			m = m.release()
		}
		return m, m.deps.Clock.Tick(FrameInterval)
	case tea.MouseMsg:
		if m.phase == confirming && msg.Action == tea.MouseActionPress {
			return m.cancelRelease(), nil // a click only cancels the confirmation
		}
		if !m.tankShown() {
			if m.tank != nil {
				m.tank.LoseHand()
			}
			return m, nil
		}
		// The window's top-left cell is the Tank's, so the two share coordinates.
		if _, h := tankArea(m.width, m.height); msg.Y < h {
			m.tank.MoveHand(msg.X, msg.Y)
		} else {
			m.tank.LoseHand() // over the status line
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// A click on the Fish is a Pat; anywhere else it drops Food.
			if render.FishAt(m.tank.Snapshot(), m.look(), msg.X, msg.Y) {
				m.tank.Pat()
			} else {
				m.tank.Drop(m.foodKind, msg.X, msg.Y)
			}
		}
	case tea.KeyMsg:
		if m.refusal != "" {
			return m, tea.Quit
		}
		switch m.phase {
		case picking, pickingForSaved:
			return m.updatePicking(msg)
		case naming:
			return m.updateNaming(msg)
		case confirming:
			return m.updateConfirming(msg)
		case farewell:
			if msg.Type == tea.KeyCtrlC {
				return m.quit()
			}
			return m, nil // the Fish is leaving: nothing else stops it
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
			if m.tankShown() {
				m.foodKind = fish.FoodKinds[key[0]-'1']
			}
		case "r":
			if m.tankShown() {
				m.phase = confirming
			}
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

// tankShown reports whether the Fish is in its open Tank and the Tank is
// drawn, so Food can be dropped into it.
func (m Model) tankShown() bool {
	return m.hasFish() && m.phase == inTank && m.tank != nil && m.width >= MinTankWidth && m.height >= MinTankHeight
}

// look is how the Fish is drawn now.
func (m Model) look() render.Look {
	return render.Look{Art: m.species().Art(), Mood: m.shown().Mood(), Monochrome: m.deps.Monochrome}
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
	case m.phase == naming:
		return render.Naming(m.width, m.height, string(m.name), m.nameErr)
	case m.width < MinTankWidth || m.height < MinTankHeight:
		return render.TooSmall(m.width, m.height)
	}
	sc := render.Scene{
		Tank: m.tank.Snapshot(),
		Look: m.look(),
		Status: render.Status{
			Name:       m.fish.Name,
			Hunger:     m.fish.Hunger,
			Age:        m.fish.Age(m.now),
			Food:       m.foodKind,
			SaveFailed: m.saveErr != nil,
		},
	}
	switch m.phase {
	case picking, pickingForSaved:
		sc.Bottom = render.Picker{Species: m.species()}
	case confirming:
		sc.Bottom = render.Confirmation{}
	case farewell:
		sc.Bottom = render.Farewell{}
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
		if m.phase == picking {
			return m.startNaming(m.species()), nil // a new Fish is named next
		}
		// A Fish saved before Species existed opens its Tank now.
		m.fish.Species, m.phase = m.species(), inTank
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
		f, err := fish.Hatch(string(m.name), m.chosenSpecies, m.deps.Clock.Now())
		if err != nil {
			m.name, m.nameErr = nil, err
			return m, nil
		}
		m.fish, m.phase = f, inTank
		if m.tank != nil {
			m.tank.Clear() // nothing is left over from a released Fish
		}
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

// hasFish reports whether a Fish is living in this Tank: it has a Species,
// has been named and loaded without error, and this launch holds the lock.
// A refused launch has no Fish of its own, so it never saves.
func (m Model) hasFish() bool {
	return m.refusal == "" && m.phase.live() && m.err == nil
}

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

// updateConfirming handles a key while the user is asked whether to release
// the Fish: y confirms, and the farewell starts, and Ctrl-C quits. Every
// other key, q and Esc included, only cancels.
func (m Model) updateConfirming(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m.quit()
	case "y":
		return m.sayBye(), nil
	}
	return m.cancelRelease(), nil
}

// cancelRelease takes down the confirmation, back to the Fish in its Tank.
func (m Model) cancelRelease() Model {
	m.phase = inTank
	return m
}

// sayBye starts the farewell, once the user has confirmed: the Fish swims out
// of its Tank, and the Tank stays live until the farewell ends and release
// lets the Fish go.
func (m Model) sayBye() Model {
	m.phase, m.byeUntil = farewell, m.deps.Clock.Now().Add(FarewellDuration)
	m.greetUntil = time.Time{} // the Tank says bye now, not welcome back
	if m.tank != nil {
		m.tank.Leave(FarewellSwim)
	}
	return m
}

// release lets the Fish go at the end of its farewell, and the Species
// picker opens for a new Fish, as on a first launch. The released Fish stays
// saved until the new one is named, so quitting before then keeps it.
func (m Model) release() Model {
	if m.tank != nil {
		m.tank.Clear() // the picker shows a new Fish in a fresh Tank
	}
	return m.startPicking(picking)
}
