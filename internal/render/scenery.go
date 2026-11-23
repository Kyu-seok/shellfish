package render

import (
	"math"

	"github.com/Kyu-seok/shellfish/internal/tank"
)

// The Scenery's colours: calm and fixed, so the Fish stays the focal point.
const (
	surfaceColour  = "#5DADE2"
	sandColour     = "#C2A878"
	darkSandColour = "#9C8460"
	pebbleColour   = "#8E8E8E"
	shellColour    = "#F4A6A6"
	kelpColour     = "#3FA34D"
	kelpTipColour  = "#6CC56F"
	grassColour    = "#8BD17C"
	bubbleColour   = "#BFEFFF"
)

// drawSurface draws the top row as a slowly drifting wave line, in its
// colour unless monochrome: ripples drifting one way over a swell drifting
// the other.
func drawSurface(g grid, s tank.Snapshot, monochrome bool) {
	for x := range s.Width {
		v := math.Sin(float64(x)*RippleLength+s.Clock*RippleSpeed) +
			SwellHeight*math.Sin(float64(x)*SwellLength-s.Clock*SwellSpeed)
		c := '~'
		switch {
		case v < WaveTrough:
			c = '_'
		case v < WaveCrest:
			c = '-'
		}
		g.setColoured(x, 0, c, plainInk, surfaceColour, monochrome)
	}
}

// drawSand draws the floor row as textured sand with pebbles, in its
// colours unless monochrome. Each column's grain is fixed, so the sand
// never shimmers.
func drawSand(g grid, s tank.Snapshot, monochrome bool) {
	floor := s.Height - 1
	for x := range s.Width {
		c, fg := '░', sandColour
		switch grain(x) % SandGrains {
		case 0:
			c, fg = '▒', darkSandColour
		case 1, 2:
			c, fg = '.', darkSandColour
		case 3:
			c, fg = ',', darkSandColour
		case 4:
			c, fg = 'o', pebbleColour
		}
		g.setColoured(x, floor, c, plainInk, fg, monochrome)
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

// drawShells draws the shells resting on the sand, in their colour unless
// monochrome.
func drawShells(g grid, s tank.Snapshot, monochrome bool) {
	for i, x := range s.Shells {
		g.setColoured(x, s.Height-2, shellGlyphs[i%len(shellGlyphs)], plainInk, shellColour, monochrome)
	}
}

// drawStalk draws a stalk of seaweed standing on the floor row, in its
// colours unless monochrome: its kelp, and its tuft of grass beside it. Kelp
// is a two-column zigzag of ( and ) that leans more the higher it is, so its
// base stays put while its tip sways. Grass is a \|/ tuft whose blade on top
// leans as it sways.
func drawStalk(g grid, st tank.Stalk, floor int, monochrome bool) {
	draw := func(x, y int, c rune, colour string) { g.setColoured(x, y, c, plainInk, colour, monochrome) }
	lean := math.Sin(2 * math.Pi * st.Sway)
	for i := range st.Height {
		off := 0
		if st.Height > 1 {
			k := float64(i) / float64(st.Height-1)
			off = int(max(-1, min(1, math.Round(KelpLean*k*k*lean))))
		}
		x, c := st.KelpCol+1+off, '('
		if i%2 == 1 {
			x, c = x+1, ')'
		}
		colour := kelpColour
		if i == st.Height-1 {
			colour = kelpTipColour
		}
		draw(x, floor-1-i, c, colour)
	}
	tuft := st.TuftCol
	for i, c := range `\|/` {
		draw(tuft+i, floor-1, c, grassColour)
	}
	if st.Tuft > 1 {
		x, c := tuft+1, '|'
		switch {
		case lean < -GrassBend:
			x, c = tuft, '\\'
		case lean > GrassBend:
			x, c = tuft+2, '/'
		}
		draw(x, floor-2, c, grassColour)
	}
}
