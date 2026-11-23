// Package render draws a snapshot of the Tank as screen text.
package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Kyu-seok/shellfish/internal/fish"
	"github.com/Kyu-seok/shellfish/internal/tank"
)

const sand = '░'

// foodGlyphs is how each Food Kind is drawn.
var foodGlyphs = map[fish.FoodKind]rune{
	fish.Pellet: '•',
	fish.Flake:  '*',
	fish.Worm:   'ʃ',
}

// FoodGlyph is how Food of kind k is drawn, so it can be shown elsewhere.
func FoodGlyph(k fish.FoodKind) string { return string(foodGlyphs[k]) }

// EatingFace is the Fish's face, whatever its Mood, just before it eats.
const EatingFace = "◕ o"

// BlinkFace is the Fish's face, whatever its Mood, while it blinks.
const BlinkFace = "- ᴗ"

// bubbleGlyphs are a bubble's sizes, smallest first.
var bubbleGlyphs = [3]rune{'°', 'o', 'O'}

// flourishGlyphs is how each heart or note is drawn.
var flourishGlyphs = map[tank.FlourishKind]rune{
	tank.Heart: '♥',
	tank.Note:  '♪',
}

// Look is how the Fish looks right now.
type Look struct {
	// Face is its eye and mouth for its Mood as it faces right, such as
	// "◕ ᴗ". It gives way to EatingFace just before the Fish eats, then to
	// BlinkFace while it blinks.
	Face string
	// Colour is the Fish's colour, such as "#FFB3BA". Empty draws it
	// without colour.
	Colour string
}

// fishSprite is the puffball facing right, wearing face, with tail as its
// first two characters. Spaces around the drawing are see-through. It fits in
// tank.FishWidth cells.
func fishSprite(face, tail string) [tank.FishHeight]string {
	return [tank.FishHeight]string{
		"   .-~~-. ",
		tail + "(  " + face + " )",
		"   `-..-' ",
	}
}

// tails are the tail's two frames facing right. It swishes between them.
var tails = [2]string{"><", ">-"}

// Tank draws the Tank as rows of exactly s.Width cells: water, bubbles,
// hearts and notes, seaweed, Food and the Fish above a sandy floor, each in
// front of the ones before it.
func Tank(s tank.Snapshot, look Look) []string {
	if s.Width <= 0 || s.Height <= 0 {
		return nil
	}
	g := newGrid(s.Width, s.Height)
	floor := s.Height - 1
	for x := range s.Width {
		g.set(x, floor, sand, plainInk)
	}
	for _, b := range s.Bubbles {
		g.set(b.Col, b.Row, bubbleGlyphs[b.Size], plainInk)
	}
	for _, fl := range s.Flourishes {
		in := fishInk
		if fl.Fading {
			in = faintInk
		}
		g.set(fl.Col, fl.Row, flourishGlyphs[fl.Kind], in)
	}
	for _, st := range s.Seaweed {
		for i := range st.Height {
			// Alternate rows lean left and right, and swap as it sways.
			x, c := st.Col, '{'
			if (i%2 == 0) != st.Sway {
				x, c = st.Col+1, '}'
			}
			g.set(x, floor-1-i, c, plainInk)
		}
	}
	for _, fd := range s.Food {
		in := plainInk
		if fd.Fading {
			in = faintInk
		}
		g.set(fd.Col, fd.Row, foodGlyphs[fd.Kind], in)
	}
	drawFish(g, s.Fish, face(s.Fish, look.Face))
	style := lipgloss.NewStyle()
	if look.Colour != "" {
		style = style.Foreground(lipgloss.Color(look.Colour))
	}
	return g.rows(style)
}

// face is the face the Fish wears: eating comes first, then blinking, then
// its Mood.
func face(f tank.Fish, mood string) string {
	switch {
	case f.MouthOpen:
		return EatingFace
	case f.Blinking:
		return BlinkFace
	}
	return mood
}

func drawFish(g grid, f tank.Fish, face string) {
	tail := tails[0]
	if f.TailUp {
		tail = tails[1]
	}
	for i, line := range fishSprite(face, tail) {
		if f.FacingLeft {
			line = mirror(line)
		}
		// Spaces around the drawing are see-through; spaces inside it are not.
		body := strings.TrimLeft(line, " ")
		x := f.Col + len(line) - len(body)
		for _, c := range strings.TrimRight(body, " ") {
			g.set(x, f.Row+i, c, fishInk)
			x += runeWidth(c)
		}
	}
}

// mirrored swaps characters for their left-right reflection.
var mirrored = map[rune]rune{'(': ')', ')': '(', '<': '>', '>': '<', '`': '\'', '\'': '`'}

// mirror flips a row of the drawing left to right.
func mirror(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	for i, c := range r {
		if m, ok := mirrored[c]; ok {
			r[i] = m
		}
	}
	return string(r)
}

func runeWidth(c rune) int { return max(ansi.StringWidth(string(c)), 1) }

// grid is the screen, one cell per terminal cell. A two-cell character
// fills its cell and leaves the next one empty.
type grid [][]cell

type cell struct {
	c  rune // 0 for the second half of a two-cell character
	in ink
}

// ink is how a cell is styled.
type ink int

const (
	plainInk ink = iota // plain
	fishInk             // part of the Fish, drawn in its colour
	faintInk            // fading away
)

var faintStyle = lipgloss.NewStyle().Faint(true)

func newGrid(w, h int) grid {
	g := make(grid, h)
	for y := range g {
		g[y] = make([]cell, w)
		for x := range g[y] {
			g[y][x].c = ' '
		}
	}
	return g
}

// set draws c at x, y in ink in, clipping anything that doesn't fit.
func (g grid) set(x, y int, c rune, in ink) {
	w := runeWidth(c)
	if y < 0 || y >= len(g) || x < 0 || x+w > len(g[y]) {
		return
	}
	g[y][x] = cell{c: c, in: in}
	if w == 2 {
		g[y][x+1] = cell{in: in}
	}
}

// rows renders the grid, styling each run of Fish cells with fishStyle.
func (g grid) rows(fishStyle lipgloss.Style) []string {
	styles := map[ink]lipgloss.Style{fishInk: fishStyle, faintInk: faintStyle}
	out := make([]string, len(g))
	for y, row := range g {
		var b strings.Builder
		for x := 0; x < len(row); {
			var run strings.Builder
			end := x
			for ; end < len(row) && row[end].in == row[x].in; end++ {
				if row[end].c != 0 {
					run.WriteRune(row[end].c)
				}
			}
			if style, ok := styles[row[x].in]; ok {
				b.WriteString(style.Render(run.String()))
			} else {
				b.WriteString(run.String())
			}
			x = end
		}
		out[y] = b.String()
	}
	return out
}
