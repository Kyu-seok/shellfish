package app_test

import (
	"math/rand/v2"
	"slices"
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
	seed    uint64
	dataDir string
	mono    bool
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

// monochrome launches as on a terminal with NO_COLOR set or too few colours.
func monochrome() option { return func(h *harness) { h.mono = true } }

// at launches with the fake clock at t.
func at(t time.Time) option { return func(h *harness) { h.clock.now = t } }

// withSeed seeds the random source, so a test can try several Tanks.
func withSeed(seed uint64) option { return func(h *harness) { h.seed = seed } }

func newHarness(t *testing.T, opts ...option) *harness {
	t.Helper()
	lipgloss.SetColorProfile(termenv.TrueColor)
	h := &harness{t: t, clock: &fakeClock{now: start}, seed: 1}
	for _, o := range opts {
		o(h)
	}
	if h.dataDir == "" {
		h.dataDir = t.TempDir()
	}
	m := app.New(app.Deps{
		Clock:      h.clock,
		Rand:       rand.New(rand.NewPCG(h.seed, 2)),
		DataDir:    h.dataDir,
		Monochrome: h.mono,
	})
	h.model, h.last = m, m.Init()
	t.Cleanup(func() { h.model.(app.Model).Close() })
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

// exitStatus is the status the shellfish process would exit with once the
// app quits, as main reads it.
func (h *harness) exitStatus() int {
	return h.model.(app.Model).ExitStatus()
}

// sighting is where the Fish was seen in one rendered frame.
type sighting struct {
	left, top  int  // top-left cell of the Fish's box
	facingLeft bool // the drawing is mirrored
	tail       string
	face       string // as it would face right, such as "◕ ᴗ"
	intact     bool   // all three rows of the drawing are whole and in place
}

// fishBoxWidth is how many cells wide the box the Fish is drawn in is.
const fishBoxWidth = 11

// The Fish's back and belly, the same either way it faces.
const (
	fishBack  = ".-~~-."
	fishBelly = "`-..-'"
)

// cells splits a row into its terminal cells. A two-cell character fills
// its first cell and leaves "" in the second.
func cells(row string) []string {
	var out []string
	for _, c := range row {
		out = append(out, string(c))
		if ansi.StringWidth(string(c)) == 2 {
			out = append(out, "")
		}
	}
	return out
}

// cellsMatch reports whether row holds want starting at cell x.
func cellsMatch(row []string, x int, want string) bool {
	w := cells(want)
	if x < 0 || x+len(w) > len(row) {
		return false
	}
	return slices.Equal(row[x:x+len(w)], w)
}

// findFish looks for the Fish's back and reads the drawing below it: its
// face between its fins, its belly and its tail.
func findFish(rows []string) (sighting, bool) {
	for y := 0; y+2 < len(rows); y++ {
		top := cells(rows[y])
		for x := range top {
			if !cellsMatch(top, x, fishBack) {
				continue
			}
			mid, low := cells(rows[y+1]), cells(rows[y+2])
			if x < 1 || mid[x-1] != "(" {
				continue
			}
			// Between its fins: the face, then a space and ")".
			end := slices.Index(mid[x:], ")")
			if end < 0 {
				continue
			}
			inner := strings.Join(mid[x:x+end], "")
			s := sighting{top: y}
			var tailAt int
			if x >= 3 && mid[x-3] == ">" && (mid[x-2] == "<" || mid[x-2] == "-") {
				s.left, tailAt = x-3, x-3
				s.intact = strings.HasPrefix(inner, "  ") && strings.HasSuffix(inner, " ") && !strings.HasSuffix(inner, "  ")
				s.face = strings.TrimSpace(inner)
			} else {
				s.facingLeft, s.left, tailAt = true, x-1, x+end+1
				s.intact = strings.HasPrefix(inner, " ") && !strings.HasPrefix(inner, "  ") && strings.HasSuffix(inner, "  ")
				f := strings.Fields(inner)
				slices.Reverse(f)
				s.face = strings.Join(f, " ")
			}
			s.intact = s.intact && cellsMatch(low, x, fishBelly)
			if tailAt+2 <= len(mid) {
				s.tail = mid[tailAt] + mid[tailAt+1]
			}
			return s, true
		}
	}
	return sighting{}, false
}

// floorRow is the index of the sandy floor: the last row that is all sand.
func floorRow(rows []string) int {
	for y := len(rows) - 1; y >= 0; y-- {
		if r := strings.TrimRight(rows[y], " "); r != "" && strings.Trim(r, "░") == "" {
			return y
		}
	}
	return -1
}

// seaweedCells counts the seaweed drawn in a frame.
func seaweedCells(rows []string) int {
	n := 0
	for _, row := range rows {
		n += strings.Count(row, "{") + strings.Count(row, "}")
	}
	return n
}

// watch advances d, one frame at a time, and returns where the Fish was seen
// in every frame. It fails the test if the Fish is ever missing, broken,
// outside the Tank, on or below the floor, or hiding any seaweed.
func (h *harness) watch(d time.Duration) []sighting {
	h.t.Helper()
	var seen []sighting
	weed := seaweedCells(h.view())
	for range int(d / app.FrameInterval) {
		h.advance(app.FrameInterval)
		rows := h.view()
		s, ok := findFish(rows)
		switch {
		case !ok:
			h.t.Fatalf("frame %d: no Fish in the Tank:\n%s", len(seen), strings.Join(rows, "\n"))
		case !s.intact:
			h.t.Fatalf("frame %d: the Fish is cut off or overlapped:\n%s", len(seen), strings.Join(rows, "\n"))
		case s.top+3 > floorRow(rows):
			h.t.Fatalf("frame %d: the Fish reaches the floor:\n%s", len(seen), strings.Join(rows, "\n"))
		case seaweedCells(rows) != weed:
			h.t.Fatalf("frame %d: the Fish covers seaweed:\n%s", len(seen), strings.Join(rows, "\n"))
		case s.tail != "><" && s.tail != ">-" && s.tail != "-<":
			h.t.Fatalf("frame %d: the Fish's tail is %q:\n%s", len(seen), s.tail, strings.Join(rows, "\n"))
		}
		seen = append(seen, s)
	}
	return seen
}

// click presses and releases the left mouse button on the cell at col, row,
// counted from the window's top-left corner, as a terminal with mouse
// reporting sends it.
func (h *harness) click(col, row int) {
	h.send(
		tea.MouseMsg{X: col, Y: row, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		tea.MouseMsg{X: col, Y: row, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft},
	)
}

// cellAt is the character drawn at col, row, or "" outside the window.
func (h *harness) cellAt(col, row int) string {
	rows := h.view()
	if row < 0 || row >= len(rows) {
		return ""
	}
	cells := []rune(rows[row])
	if col < 0 || col >= len(cells) {
		return ""
	}
	return string(cells[col])
}
