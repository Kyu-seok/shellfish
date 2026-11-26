package app_test

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/Kyu-seok/shellfish/internal/app"
	"github.com/Kyu-seok/shellfish/internal/fish"
	"github.com/Kyu-seok/shellfish/internal/tank"
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
	species fish.Species // the Species named chooses, and the harness looks for
	model   tea.Model
	last    tea.Cmd

	afterLaunch []func()
}

type option func(*harness)

// withDataDir launches on an existing data folder, as a later launch would.
func withDataDir(dir string) option { return func(h *harness) { h.dataDir = dir } }

// named launches, chooses the Fish's Species (a clownfish, unless as says
// otherwise) and names the Fish, so the Tank is open when the test starts.
func named(name string) option {
	return func(h *harness) {
		h.afterLaunch = append(h.afterLaunch, func() {
			for range slices.Index(fish.AllSpecies, h.species) {
				h.press("right")
			}
			h.press("enter")
			h.nameFish(name)
		})
	}
}

// as makes the Fish a sp: named picks it, and the harness looks for one.
func as(sp fish.Species) option { return func(h *harness) { h.species = sp } }

// monochrome launches as on a terminal with NO_COLOR set or too few colours.
func monochrome() option { return func(h *harness) { h.mono = true } }

// at launches with the fake clock at t.
func at(t time.Time) option { return func(h *harness) { h.clock.now = t } }

// withSeed seeds the random source, so a test can try several Tanks.
func withSeed(seed uint64) option { return func(h *harness) { h.seed = seed } }

func newHarness(t *testing.T, opts ...option) *harness {
	t.Helper()
	lipgloss.SetColorProfile(termenv.TrueColor)
	h := &harness{t: t, clock: &fakeClock{now: start}, seed: 1, species: fish.Clownfish}
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
// "ctrl+c", "enter", "backspace", "left", "right", " ", or any single
// printable character.
func (h *harness) press(key string) {
	switch key {
	case "left":
		h.send(tea.KeyMsg{Type: tea.KeyLeft})
	case "right":
		h.send(tea.KeyMsg{Type: tea.KeyRight})
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

// nameFish answers the naming prompt, which follows the Species picker.
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

// plain is the coloured rows raw as plain text.
func plain(raw []string) []string {
	rows := make([]string, len(raw))
	for y, r := range raw {
		rows[y] = ansi.Strip(r)
	}
	return rows
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

// sighting is where the Fish was seen in one rendered frame, and how it
// looked.
type sighting struct {
	left, top  int  // top-left cell of the Fish's box
	facingLeft bool // the drawing is mirrored: its eye is in the left half
	intact     bool // the whole box is drawn, in place
	// tailLift is how high the tail's pixels sit, on average, in pixel
	// rows from the box's top. It swings up and down as the tail swishes.
	tailLift float64

	// mood is "Content", "Hungry" or "Starving", read from its eye, or ""
	// while it blinks.
	mood     string
	blinking bool // its eye is all lid
	eating   bool // its mouth is open, just before it eats
}

// The Fish is a clownfish unless a test picks another Species: art is how
// it is drawn, and body its box and mouth in cells, from the Species table
// the app draws from.
var (
	art  = fish.Clownfish.Art()
	body = tank.BodyOf(fish.Clownfish)
)

// tearColour is the tear a Starving Fish sheds below its eye.
const tearColour = "#7fd4ff"

// isFishCell reports whether the character c is part of the Fish: the Fish
// is drawn in sextants (U+1FB00 to U+1FB3B) and ▌▐█, and nothing else is.
func isFishCell(c string) bool {
	r, _ := utf8.DecodeRuneInString(c)
	return r >= 0x1FB00 && r <= 0x1FB3B || c == "▌" || c == "▐" || c == "█"
}

// fishCells are the cells the Fish is drawn in, as col, row pairs.
func fishCells(rows []string) [][2]int {
	var out [][2]int
	for y, row := range rows {
		for x, c := range cells(row) {
			if isFishCell(c) {
				out = append(out, [2]int{x, y})
			}
		}
	}
	return out
}

// showsFish reports whether any of the Fish is drawn on screen.
func showsFish(screen string) bool {
	return len(fishCells(strings.Split(screen, "\n"))) > 0
}

// fishBounds is the box around every cell of the Fish drawn in rows, or
// false if there is none.
func fishBounds(rows []string) (minX, minY, maxX, maxY int, ok bool) {
	minX, minY, maxX, maxY = 1<<30, 1<<30, -1, -1
	for y, row := range rows {
		for x, c := range cells(row) {
			if isFishCell(c) {
				minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
			}
		}
	}
	return minX, minY, maxX, maxY, maxY >= 0
}

// behindFish reports whether a cell is inside the box around the Fish drawn
// in rows, which hides what is behind it even where its drawing shows a
// space. With no Fish drawn, nothing is.
func behindFish(rows []string) func(x, y int) bool {
	minX, minY, maxX, maxY, seen := fishBounds(rows)
	return func(x, y int) bool {
		return seen && x >= minX && x <= maxX && y >= minY && y <= maxY
	}
}

// colours is every foreground and background colour of rows' cells in
// columns x0 to x1 and rows y0 to y1.
func colours(raw []string, x0, y0, x1, y1 int) map[string]bool {
	seen := map[string]bool{}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if y < 0 || y >= len(raw) {
				continue
			}
			st := styleIn(raw[y], x)
			seen[st.fg], seen[st.bg] = true, true
		}
	}
	delete(seen, "")
	return seen
}

// eyeArea is the cells of the box the eye is drawn in, facing left or
// right, counted from the box's top-left.
func eyeArea(art fish.Art, body tank.Body, left bool) (x0, y0, x1, y1 int) {
	e := art.Eye
	x0, x1 = int(e.X-e.R), int(e.X+e.R)
	if left {
		x0, x1 = body.Width-1-x1, body.Width-1-x0
	}
	return x0, int((e.Y - e.R) / 2), x1, int((e.Y + e.R) / 2)
}

// hasColour reports whether set holds want, give or take the rounding a
// colour goes through on its way to the terminal.
func hasColour(set map[string]bool, want string) bool {
	for c := range set {
		if near(c, want) {
			return true
		}
	}
	return false
}

// near reports whether colours a and b are the same but for rounding.
func near(a, b string) bool {
	ar, ag, ab, ok := rgbOf(a)
	br, bg, bb, ok2 := rgbOf(b)
	return ok && ok2 && abs(ar-br) <= 1 && abs(ag-bg) <= 1 && abs(ab-bb) <= 1
}

// rgbOf splits a "#rrggbb" colour into its channels.
func rgbOf(colour string) (r, g, b int, ok bool) {
	if len(colour) != 7 || colour[0] != '#' {
		return 0, 0, 0, false
	}
	n, err := strconv.ParseUint(colour[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(n >> 16 & 0xff), int(n >> 8 & 0xff), int(n & 0xff), true
}

// isDark reports whether colour is a dark one, near black.
func isDark(colour string) bool {
	r, g, b, ok := rgbOf(colour)
	return ok && max(r, g, b) < 0x40
}

// find is findSpecies for the harness's Fish's Species.
func (h *harness) find(raw []string) (sighting, bool) { return findSpecies(raw, h.species) }

// body is the box and mouth of the harness's Fish's Species.
func (h *harness) body() tank.Body { return tank.BodyOf(h.species) }

// findFish finds the Fish in a frame's coloured rows and reads how it looks:
// which way its eye faces, what its eye shows and whether its mouth is open.
// The Fish is a clownfish.
func findFish(raw []string) (sighting, bool) { return findSpecies(raw, fish.Clownfish) }

// findSpecies is findFish for a Fish of Species sp.
func findSpecies(raw []string, sp fish.Species) (sighting, bool) {
	art, body := sp.Art(), tank.BodyOf(sp)
	rows := plain(raw)
	minX, minY, maxX, maxY, ok := fishBounds(rows)
	if !ok {
		return sighting{}, false
	}
	s := sighting{top: minY}
	var eye map[string]bool
	for _, left := range []bool{false, true} {
		// The head's edge is always drawn; the tail's tip may not be.
		s.left = maxX - (body.Width - 1)
		if left {
			s.left = minX
		}
		x0, y0, x1, y1 := eyeArea(art, body, left)
		eye = colours(raw, s.left+x0, s.top+y0, s.left+x1, s.top+y1)
		mono := len(colours(raw, minX, minY, maxX, maxY)) == 0 && eyeHole(rows, art, s.left, s.top, left)
		if hasColour(eye, art.Eye.White) || hasColour(eye, art.Eye.Lid) || mono {
			s.facingLeft = left
			s.intact = maxY-minY+1 == body.Height && minX >= s.left && maxX < s.left+body.Width
			if mono {
				return s, true // a silhouette: no mood, mouth or tail to read
			}
			break
		}
	}
	if !s.intact {
		return s, true
	}
	x0, y0, x1, y1 := eyeArea(art, body, s.facingLeft)
	white, lid := hasColour(eye, art.Eye.White), hasColour(eye, art.Eye.Lid)
	switch {
	case hasColour(colours(raw, s.left+x0, s.top+y0, s.left+x1, s.top+y1+1), tearColour):
		s.mood = "Starving"
	case lid && white:
		s.mood = "Hungry"
	case white:
		s.mood = "Content"
	default:
		s.blinking = true
	}
	mouth := s.left + body.MouthRight
	if s.facingLeft {
		mouth = s.left + body.MouthLeft
	}
	if st := styleIn(raw[s.top+body.MouthRow], mouth); isDark(st.fg) && (st.bg == "" || isDark(st.bg)) {
		s.eating = true
	}
	// The tail is behind where it joins the body.
	tailX0, tailX1 := 0, int(art.TailRoot)-1
	if s.facingLeft {
		tailX0, tailX1 = body.Width-1-tailX1, body.Width-1
	}
	pixels, rowSum := 0, 0
	for y := s.top; y < s.top+body.Height; y++ {
		row := cells(rows[y])
		for x := s.left + tailX0; x <= s.left+tailX1 && x < len(row); x++ {
			mask := sextantMask(row[x])
			if styleIn(raw[y], x).bg != "" {
				mask = 63 // both colours are the Fish's: no pixel is clear
			}
			for bit := range 6 {
				if mask&(1<<bit) != 0 {
					pixels++
					rowSum += (y-s.top)*3 + bit/2
				}
			}
		}
	}
	if pixels > 0 {
		s.tailLift = float64(rowSum) / float64(pixels)
	}
	return s, true
}

// filled reports whether the Fish's pixel x, y is filled, counting pixels
// from the window's top-left, 2 to a cell across and 3 down.
func filled(rows []string, x, y int) bool {
	if x < 0 || y < 0 || y/3 >= len(rows) {
		return false
	}
	row := cells(rows[y/3])
	if x/2 >= len(row) || !isFishCell(row[x/2]) {
		return false
	}
	return sextantMask(row[x/2])&(1<<(y%3*2+x%2)) != 0
}

// eyeHole reports whether the Fish drawn as art, whose box's top-left is
// left, top, has its eye as a hole in its silhouette: the eye's middle
// pixel is clear, with the Fish filled to either side of it, above and
// below.
func eyeHole(rows []string, art fish.Art, left, top int, facingLeft bool) bool {
	ex := art.Eye.X
	if facingLeft {
		ex = art.W - ex
	}
	x, y := 2*left+int(2*ex-0.5), 3*top+int(1.5*art.Eye.Y-0.5)
	if filled(rows, x, y) {
		return false
	}
	reach := func(dx, dy int) bool {
		for i := 1; i < 6; i++ {
			if filled(rows, x+i*dx, y+i*dy) {
				return true
			}
		}
		return false
	}
	return reach(-1, 0) && reach(1, 0) && reach(0, -1) && reach(0, 1)
}

// sextantMask is which of a cell's 2×3 pixels the character c fills: bits
// run left to right, top to bottom, top-left 1, top-right 2, middle-left 4,
// and so on. The sextants U+1FB00 to U+1FB3B leave out the four patterns
// with a character of their own: empty, ▌, ▐ and full.
func sextantMask(c string) int {
	switch c {
	case " ":
		return 0
	case "▌":
		return 21
	case "▐":
		return 42
	case "█":
		return 63
	}
	r, _ := utf8.DecodeRuneInString(c)
	m := int(r-0x1FB00) + 1
	if m >= 21 {
		m++
	}
	if m >= 42 {
		m++
	}
	return m
}

// fishMood is the Mood the Fish's eye shows, or "" if there is no whole
// Fish. If the Fish is blinking, it waits for the blink to end.
func (h *harness) fishMood() string {
	for range 10 {
		s, ok := h.find(h.rawView())
		if !ok || !s.intact {
			return ""
		}
		if !s.blinking {
			return s.mood
		}
		h.advance(app.FrameInterval)
	}
	return ""
}

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

// waterTop is the top row of water, just below the surface.
const waterTop = 1

// sandGlyphs are what the sandy floor is made of: sand and pebbles.
const sandGlyphs = "░▒.,o"

// floorRow is the index of the sandy floor: the last row that is all sand.
func floorRow(rows []string) int {
	for y := len(rows) - 1; y >= 0; y-- {
		if r := rows[y]; strings.Contains(r, "░") && strings.Trim(r, sandGlyphs) == "" {
			return y
		}
	}
	return -1
}

// seaweedCells counts the kelp and grass characters drawn in a frame's
// water, above the sand.
func seaweedCells(rows []string) int {
	n := 0
	for _, row := range rows[:max(floorRow(rows), 0)] {
		for _, c := range row {
			if strings.ContainsRune(`()\|/`, c) {
				n++
			}
		}
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
		raw := h.rawView()
		rows := plain(raw)
		s, ok := h.find(raw)
		switch {
		case !ok:
			h.t.Fatalf("frame %d: no Fish in the Tank:\n%s", len(seen), strings.Join(rows, "\n"))
		case !s.intact:
			h.t.Fatalf("frame %d: the Fish is cut off or overlapped (%+v):\n%s", len(seen), s, strings.Join(rows, "\n"))
		case s.top+h.body().Height > floorRow(rows):
			h.t.Fatalf("frame %d: the Fish reaches the floor:\n%s", len(seen), strings.Join(rows, "\n"))
		case seaweedCells(rows) != weed:
			h.t.Fatalf("frame %d: the Fish covers seaweed:\n%s", len(seen), strings.Join(rows, "\n"))
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

// style is how a cell is coloured: its foreground and background as
// "#rrggbb" ("" for the terminal's default) and whether it is faint.
type style struct {
	fg, bg string
	faint  bool
}

// styleAt reads the style of the character at col, row from the coloured
// view, counting columns as cellAt does. It is the zero style outside the
// window.
func (h *harness) styleAt(col, row int) style {
	rows := h.rawView()
	if row < 0 || row >= len(rows) {
		return style{}
	}
	return styleIn(rows[row], col)
}

// styleIn reads the style of the character at col in one row of the
// coloured view, or the zero style if the row is shorter.
func styleIn(s string, col int) style {
	var cur style
	n := 0
	for i := 0; i < len(s); {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) && s[j] == 'm' {
				cur = applySGR(cur, s[i+2:j])
			}
			i = j + 1
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		if n == col {
			return cur
		}
		n++
		i += size
	}
	return style{}
}

// applySGR updates st by one Select Graphic Rendition sequence's
// parameters, as a true-colour terminal reads them.
func applySGR(st style, params string) style {
	ps := strings.Split(params, ";")
	if params == "" {
		ps = []string{"0"}
	}
	for i := 0; i < len(ps); i++ {
		switch ps[i] {
		case "0":
			st = style{}
		case "2":
			st.faint = true
		case "22":
			st.faint = false
		case "39":
			st.fg = ""
		case "49":
			st.bg = ""
		case "38", "48":
			if i+4 < len(ps) && ps[i+1] == "2" {
				var rgb [3]int
				for k := range rgb {
					rgb[k], _ = strconv.Atoi(ps[i+2+k])
				}
				hex := fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2])
				if ps[i] == "38" {
					st.fg = hex
				} else {
					st.bg = hex
				}
				i += 4
			}
		}
	}
	return st
}

// statusLine is the bottom row of the Tank, trimmed.
func (h *harness) statusLine() string {
	got := h.view()
	return strings.TrimSpace(got[len(got)-1])
}

// screen is the whole rendered Tank as plain text.
func (h *harness) screen() string { return strings.Join(h.view(), "\n") }

// reopen quits h and launches the app again on its data folder, with the
// same random seed, and the clock moved by d, as if the user had come back d
// later.
func (h *harness) reopen(d time.Duration) *harness {
	h.t.Helper()
	h.press("q")
	return newHarness(h.t, withDataDir(h.dataDir), at(h.clock.now.Add(d)), withSeed(h.seed), as(h.species))
}

// count is how many times glyph is drawn in the Tank's water, above the sand.
func (h *harness) count(glyph string) int {
	rows := h.view()
	return strings.Count(strings.Join(rows[:floorRow(rows)], "\n"), glyph)
}

// meal advances up to limit, a frame at a time, until the Fish eats: until
// the saved Hunger drops. It returns the saved Hunger after the meal and the
// frames leading up to it, or fails the test if the Fish doesn't eat in time.
func (h *harness) meal(limit time.Duration) (hunger float64, frames []string) {
	h.t.Helper()
	frames, ok := h.eats(limit)
	if !ok {
		h.t.Fatalf("the Fish ate nothing in %v:\n%s", limit, h.screen())
	}
	return readSave(h.t, h.dataDir).Hunger, frames
}

// eats advances up to limit, a frame at a time, until the Fish eats: until
// the saved Hunger drops. It returns the frames up to then, and whether the
// Fish ate.
func (h *harness) eats(limit time.Duration) (frames []string, ok bool) {
	before := readSave(h.t, h.dataDir).Hunger
	for range int(limit / app.FrameInterval) {
		h.advance(app.FrameInterval)
		frames = append(frames, h.screen())
		if readSave(h.t, h.dataDir).Hunger < before {
			return frames, true
		}
	}
	return frames, false
}

// bottom is the lowest row of water, just above the sand.
func (h *harness) bottom() int { return floorRow(h.view()) - 1 }

// frames advances d, a frame at a time, and returns every frame's rows.
func (h *harness) frames(d time.Duration) [][]string {
	var out [][]string
	for range int(d / app.FrameInterval) {
		h.advance(app.FrameInterval)
		out = append(out, h.view())
	}
	return out
}

// blockSaves makes every save to h's data folder fail, by putting a folder
// where the save file goes, and returns a func that lets saves through again.
func (h *harness) blockSaves() (unblock func()) {
	h.t.Helper()
	path := filepath.Join(h.dataDir, "fish.json")
	if err := os.Remove(path); err != nil {
		h.t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "in-the-way"), 0o755); err != nil {
		h.t.Fatal(err)
	}
	return func() {
		if err := os.RemoveAll(path); err != nil {
			h.t.Fatal(err)
		}
	}
}
