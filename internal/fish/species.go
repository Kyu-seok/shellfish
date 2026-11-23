package fish

// Species is what kind of fish the Fish is. It decides how the Fish looks
// and its size, never how it swims, eats or gets hungry (ADR 0002).
type Species int

const (
	// NoSpecies is a Fish saved before Species existed: it is asked for one
	// the next time it is opened.
	NoSpecies Species = iota
	Clownfish
	Goldfish
)

// AllSpecies are every Species, in the order the picker shows them.
var AllSpecies = []Species{Clownfish, Goldfish}

// String is the Species' name, which is also how it is saved, so it never
// changes.
func (s Species) String() string { return species[s].name }

// Title is the Species' name as the picker shows it.
func (s Species) Title() string { return species[s].title }

// Art is how the Species is drawn.
func (s Species) Art() Art { return species[s].art }

// SpeciesNamed is the Species whose String is name, or false if there is
// none.
func SpeciesNamed(name string) (Species, bool) {
	for _, s := range AllSpecies {
		if s.String() == name {
			return s, true
		}
	}
	return NoSpecies, false
}

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

// species describes each Species, in the order of the Species constants:
// adding a Species means adding a constant, a row here and an entry in
// AllSpecies.
var species = []struct {
	name  string // stable, for the save file
	title string // for the picker
	art   Art
}{
	Clownfish: {name: "clownfish", title: "Clownfish", art: clownfish()},
	Goldfish:  {name: "goldfish", title: "Goldfish", art: goldfish()},
}

// clownfish is orange with three white bands edged in black.
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

// goldfish is golden orange, paler underneath, with a forked tail.
func goldfish() Art {
	const g, gd, gl, ink = "#FFA630", "#E0661B", "#FFD27A", "#3A1F08"
	body := Ellipse(10.5, 4, 6.5, 3.1)
	return Art{W: 18, H: 8, TailRoot: 5,
		Layers: []Layer{
			TailFill(gd, Polygon(5.2, 4, 0, 0.2, 1.2, 2.2, 2.6, 3.7)),
			TailFill(gd, Polygon(5.2, 4, 0, 7.8, 1.2, 5.8, 2.6, 4.3)),
			TailFill(g, Polygon(5.2, 4, 1.8, 2.6, 2.4, 4, 1.8, 5.4)),
			Fill(gd, Polygon(7, 1.6, 9, 0, 12.4, 1.4)),
			Fill(gd, Polygon(9, 6.3, 10.2, 7.9, 11.8, 6.4)),
			Fill(g, body),
			Fill(gl, Both(Ellipse(11, 5.8, 5.6, 1.6), body)),
			Fill(gl, Circle(9.6, 2.8, 0.35)), Fill(gl, Circle(11.4, 3.4, 0.35)), Fill(gl, Circle(8.4, 4.2, 0.35)),
		},
		Eye:    Eye{X: 15, Y: 3, R: 1.15, White: "#FFFFFF", Iris: ink, Lid: gd},
		MouthX: 16.8, MouthY: 4.3,
	}
}
