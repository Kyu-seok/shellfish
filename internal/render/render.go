// Package render draws everything shellfish shows: the Tank from a snapshot,
// the status line below it, the welcome-back greeting, and the naming prompt
// and other messages that take the Tank's place.
package render

import (
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

// drawTank draws the Tank on a grid of s.Width by s.Height cells: the
// surface, the sandy floor, bubbles, hearts and notes, shells, seaweed, Food
// and the Fish, each in front of the ones before it.
func drawTank(s tank.Snapshot, look Look) grid {
	g := newGrid(max(s.Width, 0), max(s.Height, 0))
	floor := s.Height - 1
	mono := look.Monochrome
	drawSurface(g, s, mono)
	drawSand(g, s, mono)
	for _, b := range s.Bubbles {
		g.setColoured(b.Col, b.Row, bubbleGlyphs[b.Size], plainInk, bubbleColour, mono)
	}
	for _, fl := range s.Flourishes {
		in := plainInk
		if fl.Fading {
			in = faintInk
		}
		g.set(fl.Col, fl.Row, flourishGlyphs[fl.Kind], in)
	}
	drawShells(g, s, mono)
	for _, st := range s.Seaweed {
		drawStalk(g, st, floor, mono)
	}
	for _, fd := range s.Food {
		in := plainInk
		if fd.Fading {
			in = faintInk
		}
		g.setColoured(fd.Col, fd.Row, fd.Kind.Glyph(), in, fd.Kind.Colour(), mono)
	}
	drawFish(g, look.Art, s.Fish, look.Mood, mono)
	return g
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

// setColoured draws c at x, y in ink in, as set does, and in its own colour
// fg unless monochrome.
func (g grid) setColoured(x, y int, c rune, in ink, fg string, monochrome bool) {
	g.set(x, y, c, in)
	if !monochrome && y >= 0 && y < len(g) && x >= 0 && x < len(g[y]) {
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
