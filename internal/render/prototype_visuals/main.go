// PROTOTYPE, throwaway. Answers: "how fine can the Fish's pixels be, and
// should its eye be pixels or a character?" Each Species is drawn once as
// shapes, then rasterised at each terminal resolution, so every resolution
// draws the same fish. Not part of shellfish; lives on the prototype/visuals
// branch only.
//
//	go run ./internal/render/prototype_visuals
//
// v view (tank / resolutions / all species)   ←/→ species   r resolution
// f eye (pixel / glyph)   m mood   e eating   s scenery   q quit
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// ---------------------------------------------------------------- grid

type cell struct {
	r      rune // 0: second half of a wide rune
	fg, bg string
}

type grid [][]cell

func newGrid(w, h int) grid {
	g := make(grid, h)
	for y := range g {
		g[y] = make([]cell, w)
		for x := range g[y] {
			g[y][x].r = ' '
		}
	}
	return g
}

func rw(r rune) int { return max(ansi.StringWidth(string(r)), 1) }

func (g grid) set(x, y int, r rune, fg, bg string) {
	w := rw(r)
	if y < 0 || y >= len(g) || x < 0 || x+w > len(g[y]) {
		return
	}
	g[y][x] = cell{r: r, fg: fg, bg: bg}
	if w == 2 {
		g[y][x+1] = cell{fg: fg, bg: bg}
	}
}

func (g grid) text(x, y int, s, fg string) {
	for _, r := range s {
		g.set(x, y, r, fg, "")
		x += rw(r)
	}
}

var profile = termenv.ColorProfile()

func seq(hex string, bg bool) string {
	if hex == "" {
		return ""
	}
	return termenv.CSI + profile.Color(hex).Sequence(bg) + "m"
}

func (g grid) String() string {
	var b strings.Builder
	for y, row := range g {
		lastFg, lastBg := "\x00", "\x00"
		for _, c := range row {
			if c.r == 0 {
				continue
			}
			if c.fg != lastFg || c.bg != lastBg {
				b.WriteString("\x1b[0m" + seq(c.fg, false) + seq(c.bg, true))
				lastFg, lastBg = c.fg, c.bg
			}
			b.WriteRune(c.r)
		}
		b.WriteString("\x1b[0m")
		if y < len(g)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// ---------------------------------------------------------------- shapes
//
// Coordinates: x in cells, y in half-rows, so one unit is about as tall as
// it is wide and circles come out round. A Species' box is W cells by H
// half-rows; it faces right.

type shape func(x, y float64) bool

func E(cx, cy, rx, ry float64) shape {
	return func(x, y float64) bool {
		dx, dy := (x-cx)/rx, (y-cy)/ry
		return dx*dx+dy*dy <= 1
	}
}

func C(cx, cy, r float64) shape { return E(cx, cy, r, r) }

func R(x0, y0, x1, y1 float64) shape {
	return func(x, y float64) bool { return x >= x0 && x <= x1 && y >= y0 && y <= y1 }
}

// P is a polygon through x0,y0, x1,y1, ...
func P(pts ...float64) shape {
	return func(x, y float64) bool {
		in := false
		n := len(pts) / 2
		for i, j := 0, n-1; i < n; j, i = i, i+1 {
			xi, yi, xj, yj := pts[2*i], pts[2*i+1], pts[2*j], pts[2*j+1]
			if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				in = !in
			}
		}
		return in
	}
}

func In(ss ...shape) shape {
	return func(x, y float64) bool {
		for _, s := range ss {
			if !s(x, y) {
				return false
			}
		}
		return true
	}
}

func Or(ss ...shape) shape {
	return func(x, y float64) bool {
		for _, s := range ss {
			if s(x, y) {
				return true
			}
		}
		return false
	}
}

func Not(s shape) shape { return func(x, y float64) bool { return !s(x, y) } }

// spikes is n little triangles standing out of an ellipse.
func spikes(cx, cy, rx, ry float64, n int, l float64) shape {
	var ss []shape
	for i := range n {
		a := 2 * math.Pi * (float64(i) + 0.5) / float64(n)
		px, py := cx+rx*math.Cos(a), cy+ry*math.Sin(a)
		tx, ty := -math.Sin(a)*0.45, math.Cos(a)*0.45
		ss = append(ss, P(px-tx, py-ty, px+math.Cos(a)*l, py+math.Sin(a)*l, px+tx, py+ty))
	}
	return Or(ss...)
}

type layer struct {
	col  string
	s    shape
	tail bool // swishes
}

func L(col string, s shape) layer    { return layer{col: col, s: s} }
func Tail(col string, s shape) layer { return layer{col: col, s: s, tail: true} }

// ---------------------------------------------------------------- species

// eye is how a Species' eye is designed, both as pixels and as a character.
type eye struct {
	x, y, r float64 // centre and radius, in shape units
	white   string  // eye white
	iris    string
	lid     string // the eyelid, usually a shade of the body
	// glyphs are the eye as a character: Content, Hungry, Starving, blink.
	glyphs [4]rune
	ink    string // the glyph's colour
}

type species struct {
	name     string
	w, h     float64 // box: cells, half-rows
	tailRoot float64 // x where the tail joins the body
	layers   []layer
	eye      eye
	mouth    [2]float64
}

var allSpecies = []species{
	clownfish(), goldfish(), pufferfish(), blueTang(), betta(),
	angelfish(), koi(), guppy(), neonTetra(), anglerfish(),
}

func clownfish() species {
	const o, od, wt, bk = "#FF8A1F", "#D9620B", "#FFFFFF", "#1A1A1A"
	body := E(10, 4, 7, 2.95)
	return species{name: "Clownfish", w: 18, h: 8, tailRoot: 4,
		layers: []layer{
			Tail(bk, P(4.2, 4, 0.2, 0.8, 1.2, 4, 0.2, 7.2)),
			Tail(o, P(4.2, 4, 0.7, 1.4, 1.7, 4, 0.7, 6.6)),
			L(bk, P(6.5, 1.6, 8.5, 0.1, 11.5, 0.2, 13.8, 1.6)),
			L(o, P(7, 1.7, 8.8, 0.7, 11.3, 0.7, 13.2, 1.7)),
			L(bk, P(8, 6.2, 9.5, 7.9, 11.6, 6.2)),
			L(o, P(8.6, 6.2, 9.6, 7.3, 11, 6.2)),
			L(bk, E(10, 4, 7.4, 3.3)),
			L(o, body),
			L(bk, In(E(5.4, 4, 0.9, 4), body)), L(wt, In(E(5.4, 4, 0.45, 4), body)),
			L(bk, In(E(9.6, 4, 1.4, 4), body)), L(wt, In(E(9.6, 4, 0.95, 4), body)),
			L(bk, In(E(13.3, 4, 1.25, 4), body)), L(wt, In(E(13.3, 4, 0.85, 4), body)),
			L(od, E(12.3, 5.3, 1.2, 0.55)),
		},
		eye:   eye{x: 15.4, y: 3.1, r: 0.95, white: wt, iris: bk, lid: od, glyphs: [4]rune{'◕', '◓', '╥', '─'}, ink: bk},
		mouth: [2]float64{16.9, 4.4},
	}
}

func goldfish() species {
	const g, gd, gl, ink = "#FFA630", "#E0661B", "#FFD27A", "#3A1F08"
	body := E(10.5, 4, 6.5, 3.1)
	return species{name: "Goldfish", w: 18, h: 8, tailRoot: 5,
		layers: []layer{
			Tail(gd, P(5.2, 4, 0, 0.2, 1.2, 2.2, 2.6, 3.7)),
			Tail(gd, P(5.2, 4, 0, 7.8, 1.2, 5.8, 2.6, 4.3)),
			Tail(g, P(5.2, 4, 1.8, 2.6, 2.4, 4, 1.8, 5.4)),
			L(gd, P(7, 1.6, 9, 0, 12.4, 1.4)),
			L(gd, P(9, 6.3, 10.2, 7.9, 11.8, 6.4)),
			L(g, body),
			L(gl, In(E(11, 5.8, 5.6, 1.6), body)),
			L(gl, C(9.6, 2.8, 0.35)), L(gl, C(11.4, 3.4, 0.35)), L(gl, C(8.4, 4.2, 0.35)),
		},
		eye:   eye{x: 15, y: 3, r: 1.15, white: "#FFFFFF", iris: ink, lid: gd, glyphs: [4]rune{'⊙', '◓', '╥', '─'}, ink: ink},
		mouth: [2]float64{16.8, 4.3},
	}
}

func pufferfish() species {
	const y, yd, cr, sp, ink = "#F5D547", "#B88A2E", "#FFF4D0", "#9C7A30", "#222222"
	body := E(8.5, 4, 5.4, 3.3)
	return species{name: "Pufferfish", w: 16, h: 8, tailRoot: 3.4,
		layers: []layer{
			Tail(yd, P(3.6, 4, 0.3, 2.2, 1, 4, 0.3, 5.8)),
			L(yd, spikes(8.5, 4, 5.4, 3.3, 14, 0.9)),
			L(y, body),
			L(cr, In(E(8.5, 6.4, 5.2, 2), body)),
			L(sp, C(6, 2.3, 0.4)), L(sp, C(8, 1.7, 0.35)), L(sp, C(7.1, 3.6, 0.35)), L(sp, C(5.2, 4.2, 0.3)),
			L("#E8836B", E(13.7, 4.7, 0.45, 0.4)),
		},
		eye:   eye{x: 11.4, y: 3, r: 1.4, white: "#FFFFFF", iris: ink, lid: yd, glyphs: [4]rune{'◉', '◓', '╥', '─'}, ink: ink},
		mouth: [2]float64{13.8, 4.7},
	}
}

func blueTang() species {
	const b, nv, yl = "#2F5BFF", "#0D1440", "#FFD23F"
	body := E(10, 4, 7, 3)
	return species{name: "Blue tang", w: 18, h: 8, tailRoot: 3.8,
		layers: []layer{
			Tail(yl, P(3.8, 4, 0.3, 1.2, 1.3, 4, 0.3, 6.8)),
			L(nv, P(5, 1.9, 9, 0.1, 14.2, 1.5)),
			L(b, P(5.8, 1.9, 9, 0.8, 13.4, 1.7)),
			L(nv, P(5, 6.2, 9, 7.9, 13.2, 6.6)),
			L(b, P(5.8, 6.2, 9, 7.2, 12.4, 6.6)),
			L(b, body),
			L(nv, In(body, E(9, 2.6, 6.2, 1.8), Not(E(9.8, 3.1, 5.2, 1.1)))),
			L(nv, In(body, E(6.3, 4.7, 2.4, 0.9))),
			L(yl, In(body, E(4.2, 4, 0.8, 0.5))),
		},
		eye:   eye{x: 15.2, y: 3.3, r: 0.9, white: "#FFFFFF", iris: nv, lid: nv, glyphs: [4]rune{'●', '◓', '╥', '─'}, ink: nv},
		mouth: [2]float64{16.9, 4.2},
	}
}

func betta() species {
	const r, rd, rp, ink = "#E0335F", "#8E1746", "#FF7AA2", "#1A0A10"
	return species{name: "Betta", w: 18, h: 8, tailRoot: 6,
		layers: []layer{
			Tail(rd, P(6.2, 4, 0, 0, 0.6, 4, 0, 8)),
			Tail(rp, P(6.2, 3.8, 0.2, 0.6, 0.9, 0.9)),
			Tail(rp, P(6.2, 4, 0.6, 3.3, 0.6, 4.3)),
			Tail(rp, P(6.2, 4.2, 0.2, 7.4, 0.9, 7.1)),
			L(rd, P(7, 2.2, 8, 0, 12, 0.3, 12.6, 2.2)),
			L(rp, P(8.4, 1.7, 9, 0.4, 9.4, 1.7)),
			L(rd, P(7, 5.8, 8, 8, 13, 7.8, 13.6, 5.8)),
			L(rp, P(9.4, 6.4, 10, 7.7, 10.4, 6.4)),
			L(r, E(11, 4, 5.6, 2.2)),
			L(rd, E(12.6, 5.2, 1, 0.5)),
		},
		eye:   eye{x: 14.8, y: 3.4, r: 0.85, white: "#FFFFFF", iris: ink, lid: rd, glyphs: [4]rune{'◕', '◔', '╥', '‿'}, ink: ink},
		mouth: [2]float64{16.4, 4},
	}
}

func angelfish() species {
	const s, k, sd, tint, ink = "#DDE3EA", "#2A2A2A", "#9AA5B4", "#F2E2B0", "#8B1E1E"
	body := E(7.5, 5, 4.3, 3.4)
	dorsal := P(4, 3, 6.6, 0, 9.2, 3.2)
	anal := P(4, 7, 6.6, 10, 9.2, 6.8)
	all := Or(body, dorsal, anal)
	return species{name: "Angelfish", w: 14, h: 10, tailRoot: 3.6,
		layers: []layer{
			Tail(sd, P(3.6, 5, 0.3, 2.6, 1.1, 5, 0.3, 7.4)),
			L(sd, dorsal), L(sd, anal),
			L(sd, P(6.6, 0.2, 4.2, 0.6, 6.6, 0.8)),
			L(s, body),
			L(tint, In(body, E(10, 4.2, 2.5, 1.6))),
			L(k, In(all, R(5.3, 0, 6.1, 10))),
			L(k, In(body, R(8.3, 0, 8.9, 10))),
		},
		eye:   eye{x: 10.4, y: 4.4, r: 0.8, white: "#FFFFFF", iris: ink, lid: sd, glyphs: [4]rune{'◕', '◓', '╥', '─'}, ink: ink},
		mouth: [2]float64{11.8, 5.2},
	}
}

func koi() species {
	const w, or, gr, ink = "#F7F7F2", "#FF5A1F", "#C4C4BE", "#333333"
	body := E(11.5, 4, 7.6, 2.8)
	return species{name: "Koi (tancho)", w: 20, h: 8, tailRoot: 4.6,
		layers: []layer{
			Tail(gr, P(4.6, 4, 0.2, 1, 1.4, 4, 0.2, 7)),
			L(gr, P(8, 1.5, 11, 0.4, 14, 1.4)),
			L(w, body),
			L(or, In(body, E(9, 2.3, 2.6, 1.3))),
			L(or, In(body, C(15.6, 2.3, 1.15))),
			L(gr, E(14.2, 6.1, 1.6, 0.6)),
			L(gr, P(18.8, 4.8, 19.8, 6, 19.1, 4.9)),
		},
		eye:   eye{x: 17.1, y: 3.5, r: 0.7, white: gr, iris: ink, lid: gr, glyphs: [4]rune{'•', '◓', '╥', '─'}, ink: ink},
		mouth: [2]float64{19, 4.3},
	}
}

func guppy() species {
	const g, gl, pk, vi, ink = "#9DB8A0", "#D7E8D9", "#FF5FA2", "#7A3CFF", "#111111"
	tail := E(3.4, 4, 3.3, 3.8)
	body := E(9.6, 4.2, 5, 1.7)
	return species{name: "Guppy", w: 16, h: 8, tailRoot: 6,
		layers: []layer{
			Tail(pk, tail),
			Tail(vi, In(tail, C(2.4, 2.8, 0.55))), Tail(vi, In(tail, C(3.2, 5.3, 0.55))), Tail(vi, In(tail, C(1.3, 4.6, 0.5))),
			L(pk, P(7, 2.9, 8.6, 1.1, 10, 2.7)),
			L(g, body),
			L(gl, In(body, E(10, 5.4, 4, 0.8))),
		},
		eye:   eye{x: 12.7, y: 3.8, r: 0.75, white: "#FFFFFF", iris: ink, lid: "#6F8C72", glyphs: [4]rune{'◕', '◓', '╥', '─'}, ink: ink},
		mouth: [2]float64{14.5, 4.3},
	}
}

func neonTetra() species {
	const s, ne, rd, fn, ink = "#D2DAE3", "#36E3FF", "#FF3B3B", "#AEB8C4", "#111111"
	body := E(9, 3, 6, 1.9)
	return species{name: "Neon tetra", w: 16, h: 6, tailRoot: 3.4,
		layers: []layer{
			Tail(fn, P(3.4, 3, 0.3, 0.8, 1.2, 3, 0.3, 5.2)),
			L(fn, P(8, 1.4, 9, 0.2, 10.5, 1.4)),
			L(s, body),
			L(rd, In(body, E(6.6, 4, 3.8, 1.1))),
			L(ne, In(body, E(9.4, 2.6, 5.4, 0.5))),
		},
		eye:   eye{x: 13.3, y: 2.7, r: 0.75, white: "#FFFFFF", iris: ink, lid: fn, glyphs: [4]rune{'●', '◓', '╥', '─'}, ink: ink},
		mouth: [2]float64{14.9, 3.1},
	}
}

func anglerfish() species {
	const b, d, belly, lure, halo = "#4B3F63", "#2E2640", "#5E5178", "#FFF27A", "#7D7440"
	body := E(9, 4.6, 6.2, 3.2)
	return species{name: "Anglerfish", w: 18, h: 8, tailRoot: 3.2,
		layers: []layer{
			Tail(d, P(3.2, 4.4, 0.3, 2.4, 0.3, 6.4)),
			L(b, body),
			L(belly, In(body, E(9, 6.9, 5.6, 1.6))),
			L(d, P(10.6, 1.8, 12.6, 0, 14.8, 0.3, 14.8, 0.8, 12.7, 0.5, 10.9, 2)),
			L(halo, C(15.3, 1, 1.25)),
			L(lure, C(15.3, 1, 0.7)),
			L("#1B1528", P(15.3, 4.4, 12.8, 5, 15, 6.4)),
			L("#F2F2F2", Or(P(13.4, 4.85, 13.7, 5.5, 14, 4.8), P(14.3, 4.7, 14.6, 5.4, 14.9, 4.6), P(13.8, 6, 14.1, 5.4, 14.4, 6.1))),
		},
		eye:   eye{x: 12.3, y: 3.1, r: 0.65, white: "#C9C2D6", iris: "#111111", lid: d, glyphs: [4]rune{'•', '◓', '╥', '─'}, ink: lure},
		mouth: [2]float64{14.6, 5.4},
	}
}

// ---------------------------------------------------------------- faces

type mood int

const (
	content mood = iota
	hungry
	starving
)

var moodNames = []string{"Content", "Hungry", "Starving"}

type face struct {
	mood     mood
	blinking bool
	eating   bool
	glyph    bool // the eye is a character, not pixels
}

// faceLayers is the eye and mouth as pixels, for this face.
func (s species) faceLayers(f face) []layer {
	e := s.eye
	disc := C(e.x, e.y, e.r)
	var ls []layer
	if f.eating {
		ls = append(ls, L("#1A1A1A", E(s.mouth[0], s.mouth[1], 0.6, 0.6)))
	}
	if f.glyph {
		return ls
	}
	if f.blinking {
		return append(ls, L(e.lid, disc), L(e.iris, In(disc, R(-99, e.y-0.2, 99, e.y+0.25))))
	}
	ix, iy := e.x+0.25*e.r, e.y+0.05*e.r
	if f.mood == starving {
		iy += 0.25 * e.r
	}
	ls = append(ls,
		L(e.white, disc),
		L(e.iris, C(ix, iy, 0.62*e.r)),
		L("#FFFFFF", C(ix-0.22*e.r, iy-0.25*e.r, 0.22*e.r)),
	)
	switch f.mood {
	case hungry:
		ls = append(ls, L(e.lid, In(disc, R(-99, -99, 99, e.y-0.15*e.r))))
	case starving:
		ls = append(ls, L(e.lid, In(disc, R(-99, -99, 99, e.y+0.1*e.r))))
		ls = append(ls, L("#7FD4FF", E(e.x-0.2*e.r, e.y+e.r+0.7, 0.32, 0.55)))
	}
	return ls
}

// ---------------------------------------------------------------- raster

// resolution is how many pixels a cell is split into.
type resolution struct {
	name   string
	sx, sy int
}

var resolutions = []resolution{
	{"half-block 1×2", 1, 2},
	{"quadrant 2×2", 2, 2},
	{"sextant 2×3", 2, 3},
	{"braille 2×4", 2, 4},
}

const quadrants = " ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█"

// glyphFor is the character that fills the pixels in mask.
func glyphFor(res resolution, mask int) rune {
	full := 1<<(res.sx*res.sy) - 1
	switch {
	case mask == 0:
		return ' '
	case mask == full && res.sy != 4:
		return '█'
	}
	switch res.sy {
	case 2:
		if res.sx == 1 {
			return []rune(" ▀▄█")[mask]
		}
		return []rune(quadrants)[mask]
	case 3:
		switch mask {
		case 21:
			return '▌'
		case 42:
			return '▐'
		}
		i := mask - 1
		if mask > 21 {
			i--
		}
		if mask > 42 {
			i--
		}
		return rune(0x1FB00 + i)
	default: // braille: our bit i is row i/2, column i%2
		dots := [8]int{0x01, 0x08, 0x02, 0x10, 0x04, 0x20, 0x40, 0x80}
		b := 0
		for i := range 8 {
			if mask&(1<<i) != 0 {
				b |= dots[i]
			}
		}
		return rune(0x2800 + b)
	}
}

func rgb(hex string) (r, g, b float64) {
	if hex == "" {
		return 16, 16, 16 // roughly a dark terminal
	}
	var ri, gi, bi int
	fmt.Sscanf(hex, "#%02x%02x%02x", &ri, &gi, &bi)
	return float64(ri), float64(gi), float64(bi)
}

func dist(a, b string) float64 {
	ar, ag, ab := rgb(a)
	br, bg, bb := rgb(b)
	return (ar-br)*(ar-br) + (ag-bg)*(ag-bg) + (ab-bb)*(ab-bb)
}

// sample is the colour at x, y ("" for clear), with the tail swished.
func sample(ls []layer, x, y, swish, root float64) string {
	for i := len(ls) - 1; i >= 0; i-- {
		l := ls[i]
		sy := y
		if l.tail && x < root {
			sy = y + swish*(root-x)/root
		}
		if l.s(x, sy) {
			return l.col
		}
	}
	return ""
}

// drawFish rasterises s into g at cell x, y.
func drawFish(g grid, s species, res resolution, x, y int, swish float64, left bool, f face) {
	ls := append(append([]layer{}, s.layers...), s.faceLayers(f)...)
	cols, rows := int(math.Ceil(s.w)), int(math.Ceil(s.h/2))
	for r := range rows {
		for c := range cols {
			px := make([]string, 0, res.sx*res.sy)
			counts := map[string]int{}
			for j := range res.sy {
				for i := range res.sx {
					sx := float64(c) + (float64(i)+0.5)/float64(res.sx)
					if left {
						sx = s.w - sx
					}
					sy := (float64(r) + (float64(j)+0.5)/float64(res.sy)) * 2
					col := sample(ls, sx, sy, swish, s.tailRoot)
					px = append(px, col)
					counts[col]++
				}
			}
			// Keep the two most common colours; the rest take the nearer.
			a, b := top2(counts)
			fg, bg := a, b
			if fg == "" { // clear shows as the terminal's own background
				fg, bg = b, a
			}
			mask := 0
			for i, col := range px {
				if col != fg && col != bg {
					if dist(col, fg) <= dist(col, bg) {
						col = fg
					} else {
						col = bg
					}
				}
				if col == fg && fg != "" {
					mask |= 1 << i
				}
			}
			if mask == 0 && bg == "" {
				continue
			}
			g.set(x+c, y+r, glyphFor(res, mask), fg, bg)
		}
	}
	if f.glyph {
		e := s.eye
		ch := e.glyphs[f.mood]
		if f.blinking {
			ch = e.glyphs[3]
		}
		cx := int(e.x)
		if left {
			cx = cols - 1 - cx
		}
		cy := int(e.y / 2)
		mx := float64(int(e.x)) + 0.5
		under := sample(s.layers, mx, float64(cy)*2+1, 0, s.tailRoot)
		g.set(x+cx, y+cy, ch, e.ink, under)
	}
}

// top2 is the two most common colours, most common first. With only one,
// both are it.
func top2(counts map[string]int) (a, b string) {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(x, y string) int {
		if counts[x] != counts[y] {
			return counts[y] - counts[x]
		}
		return strings.Compare(x, y)
	})
	if len(keys) == 1 {
		return keys[0], keys[0]
	}
	return keys[0], keys[1]
}

// ---------------------------------------------------------------- scenery

type scenery int

const (
	oldScenery scenery = iota
	newScenery
	emojiScenery
)

var sceneryNames = []string{"old", "new", "emoji"}

const (
	water   = "#5DADE2"
	sandA   = "#C2A878"
	sandB   = "#9C8460"
	kelp    = "#3FA34D"
	kelpTip = "#6CC56F"
	grass   = "#8BD17C"
	shell   = "#F4A6A6"
	bubble  = "#BFEFFF"
	heart   = "#FF6B8B"
)

func hash(x int) int {
	x = (x ^ 61) ^ (x >> 16)
	x *= 9
	x ^= x >> 4
	x *= 0x27d4eb2d
	return x ^ (x >> 15)
}

type stalk struct {
	x, h  int
	grass bool
	phase float64
}

func stalks(w int) []stalk {
	var s []stalk
	hs := []int{5, 3, 6, 2, 4, 7}
	for i, x := 0, 3; x < w-3; i, x = i+1, x+17 {
		s = append(s, stalk{x: x, h: hs[i%len(hs)], grass: i%3 == 1, phase: float64(i) * 1.3})
	}
	return s
}

func drawScenery(g grid, sc scenery, t float64) {
	w, h := len(g[0]), len(g)
	floor := h - 1
	switch sc {
	case oldScenery:
		for x := range w {
			g.set(x, floor, '░', "", "")
		}
		for _, st := range stalks(w) {
			sway := int(t/0.8)%2 == 0
			for i := range st.h {
				x, c := st.x, '{'
				if (i%2 == 0) != sway {
					x, c = st.x+1, '}'
				}
				g.set(x, floor-1-i, c, "", "")
			}
		}
	case newScenery:
		// The surface: a slow drifting wave.
		for x := range w {
			v := math.Sin(float64(x)*0.45+t*1.2) + 0.6*math.Sin(float64(x)*0.17-t*0.7)
			c := '~'
			switch {
			case v < -0.6:
				c = '_'
			case v < 0.4:
				c = '-'
			}
			g.set(x, 0, c, water, "")
		}
		// The floor: textured sand with pebbles.
		for x := range w {
			c, fg := '░', sandA
			switch hash(x) % 11 {
			case 0:
				c, fg = '▒', sandB
			case 1, 2:
				c, fg = '.', sandB
			case 3:
				c, fg = ',', sandB
			case 4:
				c, fg = 'o', "#8E8E8E"
			}
			g.set(x, floor, c, fg, "")
		}
		// Shells resting on the sand.
		for i, x := range []int{w/3 + 4, w*3/4 + 2} {
			g.set(x, floor-1, []rune("@Ѡ")[i], shell, "")
		}
		for _, st := range stalks(w) {
			lean := math.Sin(t*2*math.Pi/3.2 + st.phase)
			if st.grass {
				off := int(math.Round(lean))
				g.text(st.x-1+off, floor-2, `\ | /`, grass)
				g.text(st.x, floor-1, `\|/`, grass)
				continue
			}
			for i := range st.h {
				// The base stays put; the higher, the more it leans.
				k := float64(i+1) / float64(st.h)
				x := st.x + int(math.Round(lean*1.8*k*k))
				c, fg := '(', kelp
				if i%2 == 1 {
					c, x = ')', x+1
				}
				if i == st.h-1 {
					fg = kelpTip
				}
				g.set(x, floor-1-i, c, fg, "")
			}
		}
	case emojiScenery:
		for x := range w {
			g.set(x, floor, '░', sandA, "")
		}
		for i, st := range stalks(w) {
			g.set(st.x, floor-1, []rune("🌿🪸🌱")[i%3], "", "")
			if st.h > 4 {
				g.set(st.x, floor-2, '🌿', "", "")
			}
		}
		g.set(w/3+4, floor-1, '🐚', "", "")
		g.set(w*3/4+2, floor-1, '🪨', "", "")
	}
}

// ---------------------------------------------------------------- model

type view int

const (
	tankView view = iota
	resolutionsView
	zooView
)

var viewNames = []string{"tank", "resolutions", "all species"}

type floater struct {
	x, y, born float64
	glyph      rune // 0 for a bubble
}

type model struct {
	w, h      int
	t         float64
	view      view
	species   int
	res       int
	mood      mood
	eating    bool
	glyphEye  bool
	scenery   scenery
	fx        float64
	left      bool
	floaters  []floater
	nextNote  float64
	nextBlink float64
	blinkTill float64
	rand      *rand.Rand
}

type tick time.Time

func doTick() tea.Cmd { return tea.Tick(time.Second/20, func(t time.Time) tea.Msg { return tick(t) }) }

func (m model) Init() tea.Cmd { return doTick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "right":
			m.species = (m.species + 1) % len(allSpecies)
		case "left":
			m.species = (m.species - 1 + len(allSpecies)) % len(allSpecies)
		case "v":
			m.view = (m.view + 1) % 3
		case "r":
			m.res = (m.res + 1) % len(resolutions)
		case "m":
			m.mood = (m.mood + 1) % 3
		case "e":
			m.eating = !m.eating
		case "f":
			m.glyphEye = !m.glyphEye
		case "s":
			m.scenery = (m.scenery + 1) % 3
		}
	case tick:
		const dt = 1.0 / 20
		m.t += dt
		fw := int(math.Ceil(allSpecies[m.species].w))
		speed := []float64{7, 3.5, 1.75}[m.mood]
		if m.left {
			m.fx -= speed * dt
		} else {
			m.fx += speed * dt
		}
		if m.fx > float64(m.w-fw-2) {
			m.left = true
		}
		if m.fx < 2 {
			m.left = false
		}
		if m.t >= m.nextBlink {
			m.blinkTill = m.t + 0.15
			m.nextBlink = m.t + 3 + 3*m.rand.Float64()
		}
		// Bubbles from the floor; hearts and notes from a Content Fish, as
		// the game shows them now.
		if m.rand.Float64() < 0.06 && m.h > 4 {
			m.floaters = append(m.floaters, floater{x: float64(m.rand.IntN(max(m.w, 1))), y: float64(m.h - 3), born: m.t})
		}
		if m.mood == content && m.t >= m.nextNote {
			m.nextNote = m.t + 6 + 9*m.rand.Float64()
			g := '♥'
			if m.rand.IntN(2) == 0 {
				g = '♪'
			}
			m.floaters = append(m.floaters, floater{x: m.fx + float64(fw)/2, y: m.fishY() - 1, born: m.t, glyph: g})
		}
		kept := m.floaters[:0]
		for _, f := range m.floaters {
			if f.glyph != 0 {
				f.y = max(f.y-1.2*dt, 0)
			} else {
				f.y -= 2.5 * dt
			}
			if f.y > 0.5 && !(f.glyph != 0 && m.t-f.born > 2.5) {
				kept = append(kept, f)
			}
		}
		m.floaters = kept
		return m, doTick()
	}
	return m, nil
}

func (m model) fishY() float64 {
	fh := math.Ceil(allSpecies[m.species].h / 2)
	mid := float64(m.h-1)/2 - fh/2
	if m.mood == starving {
		mid = float64(m.h-1) - fh - 3
	}
	return math.Round(mid + 0.6*math.Sin(m.t*2*math.Pi/1.6))
}

func (m model) face(glyph bool) face {
	return face{mood: m.mood, blinking: m.t < m.blinkTill, eating: m.eating, glyph: glyph}
}

// swish is how far the tail is swung right now, quicker when swimming fast.
func (m model) swish() float64 {
	rate := []float64{3, 2, 1.2}[m.mood]
	return 0.9 * math.Sin(m.t*2*math.Pi*rate/2)
}

func (m model) View() string {
	if m.w < 40 || m.h < 12 {
		return "make the terminal bigger"
	}
	g := newGrid(m.w, m.h-1)
	switch m.view {
	case tankView:
		m.drawTank(g)
	case resolutionsView:
		m.drawResolutions(g)
	case zooView:
		m.drawZoo(g)
	}
	eyeName := "pixel"
	if m.glyphEye {
		eyeName = "glyph"
	}
	bar := fmt.Sprintf(" view: %s (v)   ◀ %s ▶   res: %s (r)   eye: %s (f)   mood: %s (m)   eating: %v (e)   scenery: %s (s)   q quit ",
		viewNames[m.view], allSpecies[m.species].name, resolutions[m.res].name, eyeName, moodNames[m.mood], m.eating, sceneryNames[m.scenery])
	return g.String() + "\n" + termenv.String(ansi.Truncate(bar, m.w, "…")).Reverse().String()
}

func (m model) drawTank(g grid) {
	drawScenery(g, m.scenery, m.t)
	for _, f := range m.floaters {
		y := int(math.Round(f.y))
		if f.glyph != 0 {
			g.set(int(f.x), y, f.glyph, "", "")
			continue
		}
		risen := float64(m.h-3) - f.y
		r, fg := '°', bubble
		switch {
		case m.scenery == emojiScenery && risen > 4:
			r, fg = '🫧', ""
		case risen > 4:
			r = 'O'
		case risen > 1.5:
			r = 'o'
		}
		if m.scenery == oldScenery {
			fg = ""
		}
		g.set(int(f.x), y, r, fg, "")
	}
	s := allSpecies[m.species]
	drawFish(g, s, resolutions[m.res], int(m.fx), int(m.fishY()), m.swish(), m.left, m.face(m.glyphEye))
}

// drawResolutions shows the current Species at every resolution, with a
// pixel eye above and a glyph eye below.
func (m model) drawResolutions(g grid) {
	s := allSpecies[m.species]
	colW := int(math.Ceil(s.w)) + 4
	rows := int(math.Ceil(s.h / 2))
	g.text(1, 0, s.name+": the same shapes at each resolution", "")
	for i, res := range resolutions {
		x := 2 + i*colW
		g.text(x, 2, res.name, "")
		for j, glyph := range []bool{false, true} {
			y := 4 + j*(rows+3)
			if i == 0 {
				g.text(0, y-1, []string{"pixel eye", "glyph eye"}[j], "#888888")
			}
			drawFish(g, s, res, x, y, m.swish(), false, m.face(glyph))
		}
	}
}

// drawZoo shows every Species at the current resolution and eye.
func (m model) drawZoo(g grid) {
	x, y, rowH := 1, 0, 0
	for i, s := range allSpecies {
		w, h := int(math.Ceil(s.w))+3, int(math.Ceil(s.h/2))+2
		if x+w > m.w {
			x, y, rowH = 1, y+rowH, 0
		}
		name := s.name
		if i == m.species {
			name = "▸ " + name
		}
		g.text(x, y, name, "#888888")
		drawFish(g, s, resolutions[m.res], x, y+1, m.swish(), false, m.face(m.glyphEye))
		x += w
		rowH = max(rowH, h)
	}
}

func main() {
	if os.Getenv("PROTO_DUMP") != "" {
		profile = termenv.Ascii
		if os.Getenv("PROTO_DUMP") == "views" {
			m := model{w: 120, h: 40, scenery: newScenery, fx: 4, rand: rand.New(rand.NewPCG(1, 2))}
			for v := range 3 {
				for r := range resolutions {
					for sp := range allSpecies {
						for _, gl := range []bool{false, true} {
							m.view, m.res, m.species, m.glyphEye, m.mood = view(v), r, sp, gl, mood(sp%3)
							_ = m.View()
						}
					}
				}
			}
			m.view, m.res, m.glyphEye = zooView, 2, true
			fmt.Println(m.View())
			return
		}
		for _, s := range allSpecies {
			for _, res := range resolutions {
				g := newGrid(24, 6)
				drawFish(g, s, res, 1, 0, 0, false, face{})
				fmt.Printf("-- %s %s\n%s\n", s.name, res.name, g)
			}
		}
		return
	}
	m := model{scenery: newScenery, res: 1, fx: 4, rand: rand.New(rand.NewPCG(1, 2))}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
