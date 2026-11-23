// Package render draws everything shellfish shows: the Tank from a snapshot,
// the status line below it, the welcome-back greeting, and the naming prompt
// and other messages that take the Tank's place.
package render

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Kyu-seok/shellfish/internal/fish"
	"github.com/Kyu-seok/shellfish/internal/tank"
)

const sand = '░'

// faces is the Fish's face for each Mood, as it faces right.
var faces = map[fish.Mood]string{
	fish.Content:  "◕ ᴗ",
	fish.Hungry:   "• _",
	fish.Starving: "; ﹏",
}

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
	// Mood picks its face. The face gives way to EatingFace just before the
	// Fish eats, then to BlinkFace while it blinks.
	Mood fish.Mood
	// Colour is the Fish's colour, unless Monochrome.
	Colour fish.Colour
	// Monochrome draws without colour, for NO_COLOR or a terminal with too
	// few colours to show the Fish.
	Monochrome bool
}

// PuffballBody is the box the puffball fits in, and where its mouth is.
// Most faces make the drawing 10 cells wide; the box leaves room for a face
// with a two-cell character. The mouth is the front of the face's bracket,
// so the drawing's front edge whichever way it faces.
var PuffballBody = tank.Body{Width: 11, Height: 3, MouthRow: 1, MouthRight: 9, MouthLeft: 0}

// fishSprite is the puffball facing right, wearing face, with tail as its
// first two characters. Spaces around the drawing are see-through. It fits in
// PuffballBody.
func fishSprite(face, tail string) [3]string {
	return [3]string{
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
		drawStalk(g, st, floor, look.Monochrome)
	}
	for _, fd := range s.Food {
		in := plainInk
		if fd.Fading {
			in = faintInk
		}
		g.set(fd.Col, fd.Row, fd.Kind.Glyph(), in)
		if !look.Monochrome {
			g.colour(fd.Col, fd.Row, fd.Kind.Colour())
		}
	}
	drawFish(g, s.Fish, face(s.Fish, faces[look.Mood]))
	style := lipgloss.NewStyle()
	if !look.Monochrome {
		style = style.Foreground(lipgloss.Color(string(look.Colour)))
	}
	return g.rows(style)
}

// Seaweed colours.
const (
	kelpColour    = "#3FA34D"
	kelpTipColour = "#6CC56F"
	grassColour   = "#8BD17C"
)

// drawStalk draws a stalk of seaweed standing on the floor row, in its
// colours unless monochrome. Kelp is a two-column zigzag of ( and ) that
// leans more the higher it is, so its base stays put while its tip sways.
// Grass is a \|/ tuft whose blade on top leans as it sways.
func drawStalk(g grid, st tank.Stalk, floor int, monochrome bool) {
	draw := func(x, y int, c rune, colour string) {
		g.set(x, y, c, plainInk)
		if !monochrome {
			g.colour(x, y, colour)
		}
	}
	lean := math.Sin(2 * math.Pi * st.Sway)
	if st.Grass {
		for i, c := range `\|/` {
			draw(st.Col+i, floor-1, c, grassColour)
		}
		if st.Height > 1 {
			x, c := st.Col+1, '|'
			switch {
			case lean < -GrassBend:
				x, c = st.Col, '\\'
			case lean > GrassBend:
				x, c = st.Col+2, '/'
			}
			draw(x, floor-2, c, grassColour)
		}
		return
	}
	for i := range st.Height {
		off := 0
		if st.Height > 1 {
			k := float64(i) / float64(st.Height-1)
			off = int(max(-1, min(1, math.Round(KelpLean*k*k*lean))))
		}
		x, c := st.Col+1+off, '('
		if i%2 == 1 {
			x, c = x+1, ')'
		}
		colour := kelpColour
		if i == st.Height-1 {
			colour = kelpTipColour
		}
		draw(x, floor-1-i, c, colour)
	}
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
	fg string // the cell's own colour as #rrggbb, or "" for its ink's
}

// ink is how a cell is styled.
type ink int

const (
	plainInk ink = iota // plain
	fishInk             // part of the Fish, drawn in its colour
	faintInk            // fading away
)

// faintStyle is for things fading away and the key hint. Faint is not a
// colour, so it is kept in monochrome.
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

// colour draws the cell at x, y in its own colour fg, on top of its ink.
func (g grid) colour(x, y int, fg string) {
	if y >= 0 && y < len(g) && x >= 0 && x < len(g[y]) {
		g[y][x].fg = fg
	}
}

// rows renders the grid, styling each run of Fish cells with fishStyle and
// each cell with a colour of its own in that colour.
func (g grid) rows(fishStyle lipgloss.Style) []string {
	styles := map[ink]lipgloss.Style{fishInk: fishStyle, faintInk: faintStyle}
	out := make([]string, len(g))
	for y, row := range g {
		var b strings.Builder
		for x := 0; x < len(row); {
			var run strings.Builder
			end := x
			for ; end < len(row) && row[end].in == row[x].in && row[end].fg == row[x].fg; end++ {
				if row[end].c != 0 {
					run.WriteRune(row[end].c)
				}
			}
			style, ok := styles[row[x].in]
			if row[x].fg != "" {
				style, ok = style.Foreground(lipgloss.Color(row[x].fg)), true
			}
			if ok {
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
