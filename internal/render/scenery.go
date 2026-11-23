package render

import (
	"math"

	"github.com/Kyu-seok/shellfish/internal/tank"
)

// The Scenery's colours: calm and fixed, so the Fish stays the focal point.
const (
	surfaceColour = "#5DADE2"
	sandColour    = "#C2A878"
	darkSand      = "#9C8460"
	pebbleColour  = "#8E8E8E"
	shellColour   = "#F4A6A6"
)

// drawSurface draws the top row as a slowly drifting wave line.
func drawSurface(g grid, s tank.Snapshot, look Look) {
	for x := range s.Width {
		v := math.Sin(float64(x)*0.45+s.Clock*1.2) + 0.6*math.Sin(float64(x)*0.17-s.Clock*0.7)
		c := '~'
		switch {
		case v < -0.6:
			c = '_'
		case v < 0.4:
			c = '-'
		}
		g.set(x, 0, c, plainInk)
		if !look.Monochrome {
			g.colour(x, 0, surfaceColour)
		}
	}
}

// drawSand draws the floor row as textured sand with pebbles. Each column's
// grain is fixed, so the sand never shimmers.
func drawSand(g grid, s tank.Snapshot, look Look) {
	floor := s.Height - 1
	for x := range s.Width {
		c, fg := '░', sandColour
		switch grain(x) % 11 {
		case 0:
			c, fg = '▒', darkSand
		case 1, 2:
			c, fg = '.', darkSand
		case 3:
			c, fg = ',', darkSand
		case 4:
			c, fg = 'o', pebbleColour
		}
		g.set(x, floor, c, plainInk)
		if !look.Monochrome {
			g.colour(x, floor, fg)
		}
	}
}

// grain scrambles a column into a number that looks random but is always
// the same for that column.
func grain(x int) uint32 {
	h := uint32(x)
	h = (h ^ 61) ^ (h >> 16)
	h *= 9
	h ^= h >> 4
	h *= 0x27d4eb2d
	return h ^ (h >> 15)
}

// shellGlyphs are how the shells resting on the sand are drawn, in turn.
var shellGlyphs = []rune{'@', 'Ѡ'}

// drawShells draws the shells resting on the sand.
func drawShells(g grid, s tank.Snapshot, look Look) {
	for i, x := range s.Shells {
		c := shellGlyphs[i%len(shellGlyphs)]
		g.set(x, s.Height-2, c, plainInk)
		if !look.Monochrome {
			g.colour(x, s.Height-2, shellColour)
		}
	}
}
