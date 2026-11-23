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
	Pufferfish
	BlueTang
	Betta
	Angelfish
	Koi
	Guppy
	NeonTetra
	Anglerfish
)

// AllSpecies are every Species, in the order the picker shows them.
var AllSpecies = []Species{
	Clownfish, Goldfish, Pufferfish, BlueTang, Betta,
	Angelfish, Koi, Guppy, NeonTetra, Anglerfish,
}

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
	Clownfish:  {name: "clownfish", title: "Clownfish", art: clownfish()},
	Goldfish:   {name: "goldfish", title: "Goldfish", art: goldfish()},
	Pufferfish: {name: "pufferfish", title: "Pufferfish", art: pufferfish()},
	BlueTang:   {name: "blue tang", title: "Blue tang", art: blueTang()},
	Betta:      {name: "betta", title: "Betta", art: betta()},
	Angelfish:  {name: "angelfish", title: "Angelfish", art: angelfish()},
	Koi:        {name: "koi", title: "Koi", art: koi()},
	Guppy:      {name: "guppy", title: "Guppy", art: guppy()},
	NeonTetra:  {name: "neon tetra", title: "Neon tetra", art: neonTetra()},
	Anglerfish: {name: "anglerfish", title: "Anglerfish", art: anglerfish()},
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
	return Art{W: 17, H: 8, TailRoot: 5,
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
		Eye:    Eye{X: 15, Y: 3, R: 1.15, White: "#FFFFFF", Iris: ink, Lid: "#D98324"},
		MouthX: 16.8, MouthY: 4.3,
	}
}

// pufferfish is a round yellow fish with a cream belly, brown spots and
// little spines all over.
func pufferfish() Art {
	const y, yd, cr, sp, ink = "#F5D547", "#B88A2E", "#FFF4D0", "#9C7A30", "#222222"
	body := Ellipse(8.5, 4, 5.4, 3.3)
	return Art{W: 15, H: 8, TailRoot: 3.4,
		Layers: []Layer{
			TailFill(yd, Polygon(3.6, 4, 0.3, 2.2, 1, 4, 0.3, 5.8)),
			Fill(yd, Spikes(8.5, 4, 5.4, 3.3, 14, 0.9)),
			Fill(y, body),
			Fill(cr, Both(Ellipse(8.5, 6.4, 5.2, 2), body)),
			Fill(sp, Circle(6, 2.3, 0.4)), Fill(sp, Circle(8, 1.7, 0.35)), Fill(sp, Circle(7.1, 3.6, 0.35)), Fill(sp, Circle(5.2, 4.2, 0.3)),
			Fill("#E8836B", Ellipse(13.7, 4.7, 0.45, 0.4)),
		},
		Eye:    Eye{X: 11, Y: 2.57, R: 1.4, White: "#FFFFFF", Iris: ink, Lid: "#D9B23A"},
		MouthX: 13.8, MouthY: 4.7,
	}
}

// blueTang is royal blue with a navy swoosh and a yellow tail.
func blueTang() Art {
	const b, nv, yl = "#2F5BFF", "#0D1440", "#FFD23F"
	body := Ellipse(10, 4, 7, 3)
	return Art{W: 17, H: 8, TailRoot: 3.8,
		Layers: []Layer{
			TailFill(yl, Polygon(3.8, 4, 0.3, 1.2, 1.3, 4, 0.3, 6.8)),
			Fill(nv, Polygon(5, 1.9, 9, 0.1, 14.2, 1.5)),
			Fill(b, Polygon(5.8, 1.9, 9, 0.8, 13.4, 1.7)),
			Fill(nv, Polygon(5, 6.2, 9, 7.9, 13.2, 6.6)),
			Fill(b, Polygon(5.8, 6.2, 9, 7.2, 12.4, 6.6)),
			Fill(b, body),
			Fill(nv, Minus(Both(body, Ellipse(9, 2.6, 6.2, 1.8)), Ellipse(9.8, 3.1, 5.2, 1.1))),
			Fill(nv, Both(body, Ellipse(6.3, 4.7, 2.4, 0.9))),
			Fill(yl, Both(body, Ellipse(4.2, 4, 0.8, 0.5))),
		},
		Eye:    Eye{X: 15.2, Y: 3.3, R: 0.9, White: "#FFFFFF", Iris: nv, Lid: nv},
		MouthX: 16.9, MouthY: 4.2,
	}
}

// betta is crimson, with long flowing fins and tail.
func betta() Art {
	const r, rd, rp, ink = "#E0335F", "#8E1746", "#FF7AA2", "#1A0A10"
	return Art{W: 17, H: 8, TailRoot: 6,
		Layers: []Layer{
			TailFill(rd, Polygon(6.2, 4, 0, 0, 0.6, 4, 0, 8)),
			TailFill(rp, Polygon(6.2, 3.8, 0.2, 0.6, 0.9, 0.9)),
			TailFill(rp, Polygon(6.2, 4, 0.6, 3.3, 0.6, 4.3)),
			TailFill(rp, Polygon(6.2, 4.2, 0.2, 7.4, 0.9, 7.1)),
			Fill(rd, Polygon(7, 2.2, 8, 0, 12, 0.3, 12.6, 2.2)),
			Fill(rp, Polygon(8.4, 1.7, 9, 0.4, 9.4, 1.7)),
			Fill(rd, Polygon(7, 5.8, 8, 8, 13, 7.8, 13.6, 5.8)),
			Fill(rp, Polygon(9.4, 6.4, 10, 7.7, 10.4, 6.4)),
			Fill(r, Ellipse(11, 4, 5.6, 2.2)),
			Fill(rd, Ellipse(12.6, 5.2, 1, 0.5)),
		},
		Eye:    Eye{X: 14.8, Y: 3.4, R: 0.85, White: "#FFFFFF", Iris: ink, Lid: "#B01E4C"},
		MouthX: 16.4, MouthY: 4,
	}
}

// angelfish is a tall silver disc with tall fins, long trailing tail fins
// and two black bars. Its mouth is low on its snout, so it reaches Food on
// the floor.
func angelfish() Art {
	const s, k, sd, tint, ink = "#DDE3EA", "#2A2A2A", "#9AA5B4", "#F2E2B0", "#8B1E1E"
	body := Ellipse(9.5, 5, 4.3, 3.4)
	dorsal := Polygon(6, 3, 8.6, 0, 11.2, 3.2)
	anal := Polygon(6, 7, 8.6, 10, 11.2, 6.8)
	all := Either(body, dorsal, anal)
	return Art{W: 14, H: 10, TailRoot: 5.6,
		Layers: []Layer{
			TailFill(sd, Polygon(5.6, 5, 0.3, 2, 1.8, 5, 0.3, 8)),
			Fill(sd, dorsal), Fill(sd, anal),
			Fill(sd, Polygon(8.6, 0.2, 6.2, 0.6, 8.6, 0.8)),
			Fill(s, body),
			Fill(tint, Both(body, Ellipse(12, 4.2, 2.5, 1.6))),
			Fill(k, Both(all, Rect(7.3, 0, 8.1, 10))),
			Fill(k, Both(body, Rect(10.3, 0, 10.9, 10))),
		},
		Eye:    Eye{X: 11.9, Y: 4.4, R: 1, White: "#FFFFFF", Iris: ink, Lid: "#7F8A99"},
		MouthX: 13.4, MouthY: 6,
	}
}

// koi is a tancho koi: white, with a red crown and a red patch on its back.
func koi() Art {
	const w, or, gr, ink = "#F7F7F2", "#FF5A1F", "#C4C4BE", "#333333"
	body := Ellipse(11.5, 4, 7.6, 2.8)
	return Art{W: 19, H: 8, TailRoot: 4.6,
		Layers: []Layer{
			TailFill(gr, Polygon(4.6, 4, 0.2, 1, 1.4, 4, 0.2, 7)),
			Fill(gr, Polygon(8, 1.5, 11, 0.4, 14, 1.4)),
			Fill(w, body),
			Fill(or, Both(body, Ellipse(9, 2.3, 2.6, 1.3))),
			Fill(or, Both(body, Circle(15.6, 2.3, 1.15))),
			Fill(gr, Ellipse(14.2, 6.1, 1.6, 0.6)),
			Fill(gr, Polygon(18.8, 4.8, 19.8, 6, 19.1, 4.9)),
		},
		Eye:    Eye{X: 17.1, Y: 3.5, R: 0.85, White: "#E6EEF2", Iris: ink, Lid: "#9C9C96"},
		MouthX: 18.6, MouthY: 4.3,
	}
}

// guppy is a small grey-green fish with a big pink tail, spotted violet.
func guppy() Art {
	const g, gl, pk, vi, ink = "#9DB8A0", "#D7E8D9", "#FF5FA2", "#7A3CFF", "#111111"
	tail := Ellipse(3.4, 4, 3.3, 3.8)
	body := Ellipse(9.6, 4.2, 5, 1.7)
	return Art{W: 15, H: 8, TailRoot: 6,
		Layers: []Layer{
			TailFill(pk, tail),
			TailFill(vi, Both(tail, Circle(2.4, 2.8, 0.55))), TailFill(vi, Both(tail, Circle(3.2, 5.3, 0.55))), TailFill(vi, Both(tail, Circle(1.3, 4.6, 0.5))),
			Fill(pk, Polygon(7, 2.9, 8.6, 1.1, 10, 2.7)),
			Fill(g, body),
			Fill(gl, Both(body, Ellipse(10, 5.4, 4, 0.8))),
		},
		Eye:    Eye{X: 12.7, Y: 3.8, R: 0.95, White: "#FFFFFF", Iris: ink, Lid: "#6F8C72"},
		MouthX: 14.5, MouthY: 4.3,
	}
}

// neonTetra is a slim silver fish with a neon blue stripe over a red one.
func neonTetra() Art {
	const s, ne, rd, fn, ink = "#D2DAE3", "#36E3FF", "#FF3B3B", "#AEB8C4", "#111111"
	body := Ellipse(9, 3, 6, 1.9)
	return Art{W: 15, H: 6, TailRoot: 3.4,
		Layers: []Layer{
			TailFill(fn, Polygon(3.4, 3, 0.3, 0.8, 1.2, 3, 0.3, 5.2)),
			Fill(fn, Polygon(8, 1.4, 9, 0.2, 10.5, 1.4)),
			Fill(s, body),
			Fill(rd, Both(body, Ellipse(6.6, 4, 3.8, 1.1))),
			Fill(ne, Both(body, Ellipse(9.4, 2.6, 5.4, 0.5))),
		},
		Eye:    Eye{X: 13, Y: 2.6, R: 1.1, White: "#FFFFFF", Iris: ink, Lid: "#8A96A4"},
		MouthX: 14.5, MouthY: 3,
	}
}

// anglerfish is a dark purple deep-sea fish with a toothy jaw and a glowing
// lure on a stalk over its head.
func anglerfish() Art {
	const b, d, belly, lure, halo = "#4B3F63", "#2E2640", "#5E5178", "#FFF27A", "#7D7440"
	body := Ellipse(9, 4.6, 6.2, 3.2)
	return Art{W: 17, H: 8, TailRoot: 3.2,
		Layers: []Layer{
			TailFill(d, Polygon(3.2, 4.4, 0.3, 2.4, 0.3, 6.4)),
			Fill(b, body),
			Fill(belly, Both(body, Ellipse(9, 6.9, 5.6, 1.6))),
			Fill(d, Polygon(10.6, 1.8, 12.6, 0, 14.8, 0.3, 14.8, 0.8, 12.7, 0.5, 10.9, 2)),
			Fill(halo, Circle(15.3, 1, 1.25)),
			Fill(lure, Circle(15.3, 1, 0.7)),
			Fill("#1B1528", Polygon(15.3, 4.4, 12.8, 5, 15, 6.4)),
			Fill("#F2F2F2", Either(Polygon(13.4, 4.85, 13.7, 5.5, 14, 4.8), Polygon(14.3, 4.7, 14.6, 5.4, 14.9, 4.6), Polygon(13.8, 6, 14.1, 5.4, 14.4, 6.1))),
		},
		Eye:    Eye{X: 12.3, Y: 3.2, R: 1, White: "#C9C2D6", Iris: "#111111", Lid: "#6B5A8A"},
		MouthX: 15, MouthY: 5.4,
	}
}
