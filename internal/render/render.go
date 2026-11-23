// Package render draws a snapshot of the Tank as screen text.
package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Kyu-seok/shellfish/internal/tank"
)

const sand = '░'

// Look is how the Fish looks right now.
type Look struct {
	// Face is its eye and mouth as it faces right, such as "◕ ᴗ".
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

// Tank draws the Tank as rows of exactly s.Width cells: water, seaweed and the
// Fish above a sandy floor.
func Tank(s tank.Snapshot, look Look) []string {
	if s.Width <= 0 || s.Height <= 0 {
		return nil
	}
	g := newGrid(s.Width, s.Height)
	floor := s.Height - 1
	for x := range s.Width {
		g.set(x, floor, sand, false)
	}
	for _, st := range s.Seaweed {
		for i := range st.Height {
			// Alternate rows lean left and right, and swap as it sways.
			x, c := st.Col, '{'
			if (i%2 == 0) != st.Sway {
				x, c = st.Col+1, '}'
			}
			g.set(x, floor-1-i, c, false)
		}
	}
	drawFish(g, s.Fish, look.Face)
	style := lipgloss.NewStyle()
	if look.Colour != "" {
		style = style.Foreground(lipgloss.Color(look.Colour))
	}
	return g.rows(style)
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
			g.set(x, f.Row+i, c, true)
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
	c    rune // 0 for the second half of a two-cell character
	fish bool // part of the Fish, drawn in its colour
}

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

// set draws c at x, y, clipping anything that doesn't fit.
func (g grid) set(x, y int, c rune, fish bool) {
	w := runeWidth(c)
	if y < 0 || y >= len(g) || x < 0 || x+w > len(g[y]) {
		return
	}
	g[y][x] = cell{c: c, fish: fish}
	if w == 2 {
		g[y][x+1] = cell{fish: fish}
	}
}

// rows renders the grid, styling each run of Fish cells with fish.
func (g grid) rows(fish lipgloss.Style) []string {
	out := make([]string, len(g))
	for y, row := range g {
		var b strings.Builder
		for x := 0; x < len(row); {
			var run strings.Builder
			end := x
			for ; end < len(row) && row[end].fish == row[x].fish; end++ {
				if row[end].c != 0 {
					run.WriteRune(row[end].c)
				}
			}
			if row[x].fish {
				b.WriteString(fish.Render(run.String()))
			} else {
				b.WriteString(run.String())
			}
			x = end
		}
		out[y] = b.String()
	}
	return out
}
