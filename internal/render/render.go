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

// bubbleGlyphs are a bubble's sizes, smallest first.
var bubbleGlyphs = [3]rune{'°', 'o', 'O'}

// flourishGlyphs is how each heart or note is drawn.
var flourishGlyphs = map[tank.FlourishKind]rune{
	tank.Heart: '♥',
	tank.Note:  '♪',
}

// Look is how the Fish looks right now.
type Look struct {
	// Art is how the Fish is drawn.
	Art fish.Art
	// Mood shows in its eye.
	Mood fish.Mood
	// Monochrome draws without colour, for NO_COLOR or a terminal with too
	// few colours to show the Fish.
	Monochrome bool
}

// Tank draws the Tank as rows of exactly s.Width cells: the surface, the
// sandy floor, bubbles, hearts and notes, shells, seaweed, Food and the Fish,
// each in front of the ones before it.
func Tank(s tank.Snapshot, look Look) []string {
	if s.Width <= 0 || s.Height <= 0 {
		return nil
	}
	g := newGrid(s.Width, s.Height)
	floor := s.Height - 1
	drawSurface(g, s, look)
	drawSand(g, s, look)
	for _, b := range s.Bubbles {
		g.set(b.Col, b.Row, bubbleGlyphs[b.Size], plainInk)
		if !look.Monochrome {
			g.colour(b.Col, b.Row, bubbleColour)
		}
	}
	for _, fl := range s.Flourishes {
		in := plainInk
		if fl.Fading {
			in = faintInk
		}
		g.set(fl.Col, fl.Row, flourishGlyphs[fl.Kind], in)
	}
	drawShells(g, s, look)
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
	drawFish(g, look.Art, s.Fish, look.Mood, look.Monochrome)
	return g.rows()
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

func runeWidth(c rune) int { return max(ansi.StringWidth(string(c)), 1) }

// grid is the screen, one cell per terminal cell. A two-cell character
// fills its cell and leaves the next one empty.
type grid [][]cell

type cell struct {
	c  rune // 0 for the second half of a two-cell character
	in ink
	// fg and bg are its colours as "#rrggbb", or "" for the terminal's own.
	fg, bg string
}

// ink is how a cell is styled.
type ink int

const (
	plainInk ink = iota // plain
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

// paint draws the one-cell character c at x, y in colours fg and bg,
// clipping it if it is outside the grid.
func (g grid) paint(x, y int, c rune, fg, bg string) {
	if y >= 0 && y < len(g) && x >= 0 && x < len(g[y]) {
		g[y][x] = cell{c: c, fg: fg, bg: bg}
	}
}

// style is how c is drawn.
func (c cell) style() (lipgloss.Style, bool) {
	st := lipgloss.NewStyle()
	if c.in == faintInk {
		st = faintStyle
	}
	if c.fg != "" {
		st = st.Foreground(lipgloss.Color(c.fg))
	}
	if c.bg != "" {
		st = st.Background(lipgloss.Color(c.bg))
	}
	return st, c.in != plainInk || c.fg != "" || c.bg != ""
}

// rows renders the grid, styling each run of cells drawn alike.
func (g grid) rows() []string {
	// Styling is slow, so each style is worked out once a frame, around a
	// stand-in for the text.
	type key struct {
		in     ink
		fg, bg string
	}
	wraps := map[key][2]string{}
	wrap := func(c cell) [2]string {
		k := key{c.in, c.fg, c.bg}
		w, ok := wraps[k]
		if !ok {
			if style, styled := c.style(); styled {
				before, after, _ := strings.Cut(style.Render("\x00"), "\x00")
				w = [2]string{before, after}
			}
			wraps[k] = w
		}
		return w
	}
	out := make([]string, len(g))
	for y, row := range g {
		var b strings.Builder
		for x := 0; x < len(row); {
			w := wrap(row[x])
			b.WriteString(w[0])
			end := x
			for ; end < len(row) && row[end].in == row[x].in && row[end].fg == row[x].fg && row[end].bg == row[x].bg; end++ {
				if row[end].c != 0 {
					b.WriteRune(row[end].c)
				}
			}
			b.WriteString(w[1])
			x = end
		}
		out[y] = b.String()
	}
	return out
}
