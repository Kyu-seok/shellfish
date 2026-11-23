package render

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/Kyu-seok/shellfish/internal/fish"
	"github.com/Kyu-seok/shellfish/internal/tank"
)

// The Fish is drawn in sextants: each cell is split into 2×3 pixels, and
// each pixel is sampled from the Fish's shapes (ADR 0001).
const (
	pixelCols = 2
	pixelRows = 3
)

// on stands for every colour of a monochrome Fish.
const on = "on"

// drawFish rasterises art into g as the Fish f looks right now.
func drawFish(g grid, art fish.Art, f tank.Fish, mood fish.Mood, mono bool) {
	layers := slices.Concat(art.Layers, faceLayers(art, f, mood))
	if mono {
		// A silhouette, with the eye a hole in it so it still reads as a
		// face. A ring of body round the eye closes the hole where the eye
		// is near the edge.
		e := art.Eye
		layers = slices.Concat(art.Layers, []fish.Layer{
			fish.Fill(on, fish.Circle(e.X, e.Y, e.R+EyeRing)),
			fish.Fill("", fish.Circle(e.X, e.Y, e.R)),
		})
	}
	swish := TailSwing * math.Sin(math.Pi*f.Tail)
	cols, rows := int(math.Ceil(art.W)), int(math.Ceil(art.H/2))
	for r := range rows {
		for c := range cols {
			var px [pixelCols * pixelRows]string
			counts := map[string]int{}
			for j := range pixelRows {
				for i := range pixelCols {
					x := float64(c) + (float64(i)+0.5)/pixelCols
					if f.FacingLeft {
						x = art.W - x
					}
					y := (float64(r) + (float64(j)+0.5)/pixelRows) * 2
					col := sample(layers, x, y, swish, art.TailRoot)
					if mono && col != "" {
						col = on
					}
					px[j*pixelCols+i] = col
					counts[col]++
				}
			}
			fg, bg, mask := twoColours(px[:], counts)
			if mask == 0 && bg == "" {
				continue // all clear
			}
			if mono {
				fg, bg = "", ""
			}
			g.paint(f.Col+c, f.Row+r, sextant(mask), fg, bg)
		}
	}
}

// faceLayers are the eye and mouth, drawn over the body, for how the Fish
// looks right now.
func faceLayers(art fish.Art, f tank.Fish, mood fish.Mood) []fish.Layer {
	e := art.Eye
	disc := fish.Circle(e.X, e.Y, e.R)
	var ls []fish.Layer
	if f.MouthOpen {
		ls = append(ls, fish.Fill(MouthColour, fish.Circle(art.MouthX, art.MouthY, MouthRadius)))
	}
	if f.Blinking {
		line := fish.Rect(math.Inf(-1), e.Y-0.2, math.Inf(1), e.Y+0.25)
		return append(ls, fish.Fill(e.Lid, disc), fish.Fill(e.Iris, fish.Both(disc, line)))
	}
	ix, iy := e.X+0.25*e.R, e.Y+0.05*e.R
	if mood == fish.Starving {
		iy += 0.25 * e.R
	}
	ls = append(ls,
		fish.Fill(e.White, disc),
		fish.Fill(e.Iris, fish.Circle(ix, iy, 0.62*e.R)),
		fish.Fill(Highlight, fish.Circle(ix-0.22*e.R, iy-0.25*e.R, 0.22*e.R)),
	)
	above := func(y float64) fish.Shape {
		return fish.Both(disc, fish.Rect(math.Inf(-1), math.Inf(-1), math.Inf(1), y))
	}
	switch mood {
	case fish.Hungry:
		ls = append(ls, fish.Fill(e.Lid, above(e.Y-0.15*e.R)))
	case fish.Starving:
		ls = append(ls,
			fish.Fill(e.Lid, above(e.Y+0.1*e.R)),
			fish.Fill(TearColour, fish.Ellipse(e.X-0.2*e.R, e.Y+e.R+0.7, 0.32, 0.55)),
		)
	}
	return ls
}

// sample is the colour at x, y ("" for clear), with the tail swung by swish
// half-rows at its tip.
func sample(ls []fish.Layer, x, y, swish, root float64) string {
	for i := len(ls) - 1; i >= 0; i-- {
		l := ls[i]
		sy := y
		if l.Tail && x < root {
			sy = y + swish*(root-x)/root
		}
		if l.Shape(x, sy) {
			return l.Colour
		}
	}
	return ""
}

// twoColours fits a cell's pixels to the two colours a cell can show: the
// two covering the most pixels, the rest taking whichever of them is nearer.
// mask has a bit set for each pixel in fg. Clear pixels show the terminal's
// background, so a cell with any has no bg.
func twoColours(px []string, counts map[string]int) (fg, bg string, mask int) {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if counts[a] != counts[b] {
			return counts[b] - counts[a]
		}
		return strings.Compare(a, b)
	})
	fg, bg = keys[0], keys[0]
	if len(keys) > 1 {
		bg = keys[1]
	}
	if fg == "" {
		fg, bg = bg, fg
	}
	for i, col := range px {
		if col != fg && col != bg && distance(col, fg) <= distance(col, bg) {
			col = fg
		}
		if col == fg && fg != "" {
			mask |= 1 << i
		}
	}
	if fg == bg {
		bg = ""
	}
	return fg, bg, mask
}

// distance is how far apart two colours are in RGB, squared. Clear counts
// as a dark terminal.
func distance(a, b string) float64 {
	ar, ag, ab := rgb(a)
	br, bg, bb := rgb(b)
	return (ar-br)*(ar-br) + (ag-bg)*(ag-bg) + (ab-bb)*(ab-bb)
}

func rgb(hex string) (r, g, b float64) {
	n, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		return 16, 16, 16
	}
	return float64(n >> 16 & 0xff), float64(n >> 8 & 0xff), float64(n & 0xff)
}

// sextant is the character with the pixels in mask filled. Bits run left to
// right, top to bottom: top-left 1, top-right 2, middle-left 4 and so on.
func sextant(mask int) rune {
	switch mask {
	case 0:
		return ' '
	case 63:
		return '█'
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
}
