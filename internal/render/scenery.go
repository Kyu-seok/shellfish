package render

import (
	"math"

	"github.com/Kyu-seok/shellfish/internal/tank"
)

// The Scenery's colours: calm and fixed, so the Fish stays the focal point.
const (
	surfaceColour = "#5DADE2"
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
