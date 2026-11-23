package render

import (
	"math"

	"github.com/Kyu-seok/shellfish/internal/tank"
)

// Art is how a Species is drawn: a list of coloured shapes in a box W cells
// wide and H half-rows tall, facing right, with its eye and mouth.
type Art struct {
	W, H float64
	// TailRoot is the x where the tail joins the body. The tail swings more
	// the further behind it.
	TailRoot float64
	Layers   []Layer
	Eye      Eye
	// MouthX and MouthY are the mouth, where the Fish eats.
	MouthX, MouthY float64
}

// Eye is how a Species' eye is drawn.
type Eye struct {
	X, Y, R float64 // centre and radius
	White   string
	Iris    string
	Lid     string // the eyelid, usually a shade of the body
}

// The colours of the eye and mouth that every Species shares.
const (
	// Highlight is the glint in a Content or Hungry eye.
	Highlight = "#FFFFFF"
	// TearColour is the tear a Starving Fish sheds below its eye.
	TearColour = "#7FD4FF"
	// MouthColour is the inside of the open mouth, just before the Fish eats.
	MouthColour = "#1A1A1A"
)

// Clownfish is an orange fish with three white bands edged in black.
var Clownfish = clownfish()

func clownfish() Art {
	const o, od, wt, bk = "#FF8A1F", "#D9620B", "#FFFFFF", "#1A1A1A"
	body := Ellipse(10, 4, 7, 2.95)
	return Art{W: 18, H: 8, TailRoot: 4,
		Layers: []Layer{
			TailFill(bk, Polygon(4.2, 4, 0.2, 0.8, 1.2, 4, 0.2, 7.2)),
			TailFill(o, Polygon(4.2, 4, 0.7, 1.4, 1.7, 4, 0.7, 6.6)),
			Fill(bk, Polygon(6.5, 1.6, 8.5, 0.1, 11.5, 0.2, 13.8, 1.6)),
			Fill(o, Polygon(7, 1.7, 8.8, 0.7, 11.3, 0.7, 13.2, 1.7)),
			Fill(bk, Polygon(8, 6.2, 9.5, 7.9, 11.6, 6.2)),
			Fill(o, Polygon(8.6, 6.2, 9.6, 7.3, 11, 6.2)),
			Fill(bk, Ellipse(10, 4, 7.4, 3.3)),
			Fill(o, body),
			Fill(bk, Both(Ellipse(5.4, 4, 0.9, 4), body)), Fill(wt, Both(Ellipse(5.4, 4, 0.45, 4), body)),
			Fill(bk, Both(Ellipse(9.6, 4, 1.4, 4), body)), Fill(wt, Both(Ellipse(9.6, 4, 0.95, 4), body)),
			Fill(bk, Both(Ellipse(13.3, 4, 1.25, 4), body)), Fill(wt, Both(Ellipse(13.3, 4, 0.85, 4), body)),
			Fill(od, Ellipse(12.3, 5.3, 1.2, 0.55)),
		},
		Eye:    Eye{X: 15.4, Y: 3.1, R: 0.95, White: wt, Iris: bk, Lid: od},
		MouthX: 16.9, MouthY: 4.4,
	}
}

// Body is the box the drawing fits in, in cells, and the cell its mouth is
// in, facing either way.
func (a Art) Body() tank.Body {
	w := int(math.Ceil(a.W))
	col := int(a.MouthX)
	return tank.Body{
		Width:      w,
		Height:     int(math.Ceil(a.H / 2)),
		MouthRow:   int(a.MouthY / 2),
		MouthRight: col,
		MouthLeft:  w - 1 - col,
	}
}
