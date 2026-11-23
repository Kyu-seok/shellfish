// Package render draws a snapshot of the Tank as screen text.
package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Kyu-seok/shellfish/internal/tank"
)

const sand = '░'

// Fish drawings, row by row, without the tail. Spaces are see-through. Each
// row is tank.FishWidth cells wide, and the tail fills the first two cells
// facing right or the last two facing left.
var (
	facingRight = [tank.FishHeight]string{"   .-~~-. ", "  (  ◕ ᴗ )", "   `-..-' "}
	facingLeft  = mirror(facingRight)
)

// Tail frames, swishing between the two as the Fish swims.
var (
	tailRight = [2]string{"><", ">-"}
	tailLeft  = [2]string{mirrorRow(tailRight[0]), mirrorRow(tailRight[1])}
)

// Tank draws the Tank as rows of exactly s.Width cells: water, seaweed and the
// Fish, in fishColour (such as "#FFB3BA"), above a sandy floor.
func Tank(s tank.Snapshot, fishColour string) []string {
	if s.Width <= 0 || s.Height <= 0 {
		return nil
	}
	g := newGrid(s.Width, s.Height)
	floor := s.Height - 1
	for x := range s.Width {
		g.set(x, floor, sand)
	}
	for _, st := range s.Seaweed {
		for i := range st.Height {
			// Alternate rows lean left and right, and swap as it sways.
			x, c := st.Col, '{'
			if (i%2 == 0) != st.Sway {
				x, c = st.Col+1, '}'
			}
			g.set(x, floor-1-i, c)
		}
	}
	drawFish(g, s.Fish)
	return g.rows(lipgloss.NewStyle().Foreground(lipgloss.Color(fishColour)))
}

func drawFish(g grid, f tank.Fish) {
	sprite, tails, tailAt := facingRight, tailRight, 0
	if f.FacingLeft {
		sprite, tails, tailAt = facingLeft, tailLeft, tank.FishWidth-2
	}
	tail := tails[0]
	if f.TailUp {
		tail = tails[1]
	}
	for i, line := range sprite {
		cells := []rune(line)
		if i == 1 {
			copy(cells[tailAt:], []rune(tail))
		}
		// Spaces around the drawing are see-through; spaces inside it are not.
		first, last := len(cells), -1
		for j, c := range cells {
			if c != ' ' {
				first, last = min(first, j), j
			}
		}
		for j, c := range cells {
			if j >= first && j <= last {
				g.setFish(f.Col+j, f.Row+i, c)
			}
		}
	}
}

// mirror flips a drawing left to right.
func mirror(rows [tank.FishHeight]string) [tank.FishHeight]string {
	var out [tank.FishHeight]string
	for i, r := range rows {
		out[i] = mirrorRow(r)
	}
	return out
}

// mirrored swaps characters for their left-right reflection.
var mirrored = map[rune]rune{'(': ')', ')': '(', '<': '>', '>': '<', '`': '\'', '\'': '`'}

// mirrorRow reverses a row of single-cell characters and reflects each one.
// The face is swapped as a whole so the eye stays at the front.
func mirrorRow(s string) string {
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

// grid is the screen, one cell per character. Every character drawn is one
// cell wide.
type grid [][]cell

type cell struct {
	c    rune
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

// set draws c at x, y, clipping anything outside.
func (g grid) set(x, y int, c rune) { g.put(x, y, cell{c: c}) }

// setFish draws c at x, y as part of the Fish.
func (g grid) setFish(x, y int, c rune) { g.put(x, y, cell{c: c, fish: true}) }

func (g grid) put(x, y int, c cell) {
	if y >= 0 && y < len(g) && x >= 0 && x < len(g[y]) {
		g[y][x] = c
	}
}

// rows renders the grid, styling each run of Fish cells with fish.
func (g grid) rows(fish lipgloss.Style) []string {
	out := make([]string, len(g))
	for y, row := range g {
		var b strings.Builder
		for x := 0; x < len(row); {
			end := x
			var run []rune
			for end < len(row) && row[end].fish == row[x].fish {
				run = append(run, row[end].c)
				end++
			}
			if row[x].fish {
				b.WriteString(fish.Render(string(run)))
			} else {
				b.WriteString(string(run))
			}
			x = end
		}
		out[y] = b.String()
	}
	return out
}
