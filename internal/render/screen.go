package render

import (
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/Kyu-seok/shellfish/internal/fish"
	"github.com/Kyu-seok/shellfish/internal/tank"
)

// Scene is everything shown while the Tank is open.
type Scene struct {
	Tank   tank.Snapshot
	Look   Look
	Status Status
	// Away is how long the user was away, while the Tank greets them for
	// it. Zero shows no greeting.
	Away time.Duration
	// Bottom is what takes the status line's place, if anything: nil shows
	// the status line.
	Bottom Bottom
}

// Bottom is a line shown below the Tank in the status line's place: one of
// Picker, Confirmation and Farewell.
type Bottom interface {
	// line is the line, exactly width cells, for the Fish with status s.
	line(s Status, width int) string
}

// Picker is the Species picker, showing Species.
type Picker struct{ Species fish.Species }

// Confirmation asks the user to confirm releasing the Fish.
type Confirmation struct{}

// Farewell says bye while the released Fish swims away.
type Farewell struct{}

func (p Picker) line(_ Status, width int) string { return pickerLine(p.Species, width) }

func (Confirmation) line(s Status, width int) string {
	return fit(" Release "+s.Name+" to the sea? y/N", width)
}

func (Farewell) line(s Status, width int) string { return fit(" Bye, "+s.Name+"!", width) }

// Status is what the status line shows.
type Status struct {
	Name   string
	Hunger float64
	Age    int // in whole Days
	// Food is the selected Food Kind: what a click or f drops.
	Food fish.FoodKind
	// SaveFailed is set while the Fish can't be saved.
	SaveFailed bool
}

// Screen draws the open Tank as rows of exactly its width, with the
// greeting over the water if it is up and the status line below.
func Screen(sc Scene) string {
	g := drawTank(sc.Tank, sc.Look)
	if sc.Away > 0 {
		drawGreeting(g, greeting(sc.Status.Name, sc.Away), sc.Tank.Food)
	}
	water := g.rows()
	bottom := statusLine(sc.Status, sc.Tank.Width)
	if sc.Bottom != nil {
		bottom = sc.Bottom.line(sc.Status, sc.Tank.Width)
	}
	return strings.Join(append(water, bottom), "\n")
}

// pickerHint teaches the Species picker's keys.
const pickerHint = "←/→ browse · enter choose"

// pickerLine is exactly width cells: the Species shown on the picker on the
// left and a faint key hint at the right end, which makes way when there is
// no room.
func pickerLine(s fish.Species, width int) string {
	species := " Species: ‹ " + s.Title() + " ›"
	if gap := width - ansi.StringWidth(species) - ansi.StringWidth(pickerHint) - 1; gap >= 2 {
		return species + strings.Repeat(" ", gap) + faintStyle.Render(pickerHint) + " "
	}
	return fit(species, width)
}

// greeting welcomes the user back to the Fish called name after away.
func greeting(name string, away time.Duration) string {
	return fmt.Sprintf("%s missed you! You were gone %s.", name, readable(away))
}

// readable is an absence in whole hours, or whole Days from a Day on.
func readable(d time.Duration) string {
	if d >= fish.Day {
		return count(int(d/fish.Day), "day")
	}
	return count(int(d/time.Hour), "hour")
}

// count is n of unit, such as "1 day" or "3 days".
func count(n int, unit string) string {
	if n != 1 {
		unit += "s"
	}
	return fmt.Sprintf("%d %s", n, unit)
}

// drawGreeting draws text, centred and wrapped to fit, over the water from
// GreetingRow down. It covers only the text's own cells, never the sand,
// and never food, so Food dropped while it is up shows at once.
func drawGreeting(g grid, text string, food []tank.Food) {
	if len(g) == 0 {
		return
	}
	under := map[[2]int]bool{}
	for _, fd := range food {
		under[[2]int{fd.Col, fd.Row}] = true
	}
	width := len(g[0])
	lines := strings.Split(ansi.Wordwrap(text, width-2, ""), "\n")
	blockW := 0
	for _, l := range lines {
		blockW = max(blockW, ansi.StringWidth(l))
	}
	left := max((width-blockW)/2, 0)
	for i, line := range lines {
		y := GreetingRow + i
		if y >= len(g)-1 { // the last row is the sand
			break
		}
		x := left
		for _, c := range line {
			if !under[[2]int{x, y}] {
				g.set(x, y, c, plainInk)
			}
			x += runeWidth(c)
		}
	}
}

// keyHint teaches the controls at the right end of the status line, and
// shortKeyHint takes its place when it doesn't fit.
const (
	keyHint      = "click/f feed · 1-3 food · r release · q quit"
	shortKeyHint = "f feed · r release · q quit"
)

// saveFailed takes the key hint's place while the Fish can't be saved.
const saveFailed = "⚠ couldn't save"

// statusLine is exactly width cells: the Fish's facts on the left and a
// faint key hint at the right end, which shortens, then makes way, when there
// is no room. The selected Food Kind, last of the facts, shrinks from its
// glyph and name to its glyph alone before the hint shortens or makes way.
func statusLine(s Status, width int) string {
	glyph := string(s.Food.Glyph())
	named := glyph + " " + s.Food.String()
	hints := []string{faintStyle.Render(keyHint), faintStyle.Render(shortKeyHint)}
	if s.SaveFailed {
		hints = []string{saveFailed}
	}
	for _, hint := range hints {
		for _, food := range []string{named, glyph} {
			facts := statusFacts(s, width, food)
			if gap := width - ansi.StringWidth(facts) - ansi.StringWidth(hint) - 1; gap >= 2 {
				return facts + strings.Repeat(" ", gap) + hint + " "
			}
		}
	}
	return fit(statusFacts(s, width, named, glyph), width)
}

// statusFacts shows the Fish's name, then its Hunger meter, its Age and the
// first of foods (ways of showing the selected Food Kind) that fits, each
// only if it fits whole in width, so it never ends in a dangling "·".
func statusFacts(s Status, width int, foods ...string) string {
	facts := " " + s.Name
	for _, item := range [][]string{
		{"Hunger " + hungerMeter(s.Hunger)},
		{"Age " + count(s.Age, "day")},
		foods,
	} {
		for _, way := range item {
			if next := facts + " · " + way; ansi.StringWidth(next) <= width {
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

// Naming is the first-launch prompt asking what the Fish is called, showing
// the name typed so far and why the last one was rejected, if it was.
func Naming(width, height int, name string, rejected error) string {
	lines := []string{"What is your Fish called?", "", "> " + name}
	if rejected != nil {
		lines = append(lines, "", sentence(rejected))
	}
	return Message(width, height, lines...)
}

// Failure says why the Tank can't be opened.
func Failure(width, height int, err error) string {
	return Message(width, height, "shellfish: "+err.Error(), "", "Press q to quit.")
}

// TooSmall asks for a bigger window, in place of a Tank that doesn't fit.
func TooSmall(width, height int) string {
	msg := ansi.Wordwrap("Make me a bigger tank 🐟", width, "")
	return Message(width, height, strings.Split(msg, "\n")...)
}

// Message fills a width×height window with a block of lines centred in it.
func Message(width, height int, lines ...string) string {
	rows := make([]string, height)
	for y := range rows {
		rows[y] = fit("", width)
	}
	top := max((height-len(lines))/2, 0)
	for i, line := range centre(lines, width) {
		if y := top + i; y < height {
			rows[y] = line
		}
	}
	return strings.Join(rows, "\n")
}

// centre centres a block of lines in width cells, sharing one left edge,
// each fitted to exactly width.
func centre(lines []string, width int) []string {
	blockW := 0
	for _, l := range lines {
		blockW = max(blockW, ansi.StringWidth(l))
	}
	left := strings.Repeat(" ", max((width-blockW)/2, 0))
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = fit(left+l, width)
	}
	return out
}

// fit pads or truncates s to exactly w cells.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	return s + strings.Repeat(" ", w-ansi.StringWidth(s))
}

// sentence is err's message as a sentence: capitalised, with a full stop.
func sentence(err error) string {
	msg := err.Error()
	r, size := utf8.DecodeRuneInString(msg)
	return string(unicode.ToUpper(r)) + msg[size:] + "."
}
