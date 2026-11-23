package app_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Kyu-seok/shellfish/internal/app"
)

// plant is one seaweed cell seen in the Tank.
type plant struct {
	col, row int
	glyph    string
	fg       string
}

const (
	kelpGlyphs  = "()"
	grassGlyphs = `\|/`
)

// plants is every seaweed cell drawn in a green, above the sand.
func (h *harness) plants() []plant {
	rows := h.view()
	var out []plant
	for y := range floorRow(rows) {
		for x, c := range []rune(rows[y]) {
			g := string(c)
			if !strings.Contains(kelpGlyphs+grassGlyphs, g) {
				continue
			}
			if fg := h.styleAt(x, y).fg; green(fg) {
				out = append(out, plant{col: x, row: y, glyph: g, fg: fg})
			}
		}
	}
	return out
}

// rgb splits a "#rrggbb" colour into its channels.
func rgb(hex string) (r, g, b int, ok bool) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0, false
	}
	n, err := strconv.ParseUint(hex[1:], 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(n >> 16), int(n >> 8 & 0xff), int(n & 0xff), true
}

// green reports whether hex is a colour whose green clearly leads.
func green(hex string) bool {
	r, g, b, ok := rgb(hex)
	return ok && g > r+30 && g > b+30
}

// brightness is a rough lightness of hex.
func brightness(hex string) int {
	r, g, b, _ := rgb(hex)
	return r + g + b
}

func TestTheTankGrowsGreenKelpWithALighterTipAndGrassTufts(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(120, 30)

	var kelp, grass []plant
	for _, p := range h.plants() {
		if strings.Contains(kelpGlyphs, p.glyph) {
			kelp = append(kelp, p)
		} else {
			grass = append(grass, p)
		}
	}
	if len(kelp) == 0 || len(grass) == 0 {
		t.Fatalf("found %d green kelp cells and %d green grass cells, want some of each:\n%s", len(kelp), len(grass), h.screen())
	}

	// Each stalk's top cell is its tip, lighter than the kelp below it.
	bottom := floorRow(h.view()) - 1
	var stalkFg, tipFg string
	for _, p := range kelp {
		if p.row == bottom {
			stalkFg = p.fg
		}
	}
	for _, p := range kelp {
		below := false
		for _, q := range kelp {
			if q.row == p.row+1 && q.col >= p.col-2 && q.col <= p.col+2 {
				below = true
			}
		}
		above := false
		for _, q := range kelp {
			if q.row == p.row-1 && q.col >= p.col-2 && q.col <= p.col+2 {
				above = true
			}
		}
		if below && !above {
			tipFg = p.fg
		}
	}
	if stalkFg == "" || tipFg == "" || brightness(tipFg) <= brightness(stalkFg) {
		t.Errorf("kelp is %q with tip %q, want a lighter green tip:\n%s", stalkFg, tipFg, h.screen())
	}
}

func TestSeaweedHasNoColourInMonochrome(t *testing.T) {
	h := newHarness(t, named("Bubbles"), monochrome())
	h.resize(120, 30)
	h.advance(time.Second)
	rows := h.view()
	for y := range floorRow(rows) {
		for x, c := range []rune(rows[y]) {
			if strings.ContainsRune(kelpGlyphs+grassGlyphs, c) && h.styleAt(x, y).fg != "" {
				t.Fatalf("seaweed %q at (%d, %d) is coloured %q in monochrome:\n%s", c, x, y, h.styleAt(x, y).fg, h.screen())
			}
		}
	}
	if !strings.ContainsAny(strings.Join(rows[:floorRow(rows)], ""), kelpGlyphs+grassGlyphs) {
		t.Fatalf("no seaweed in monochrome:\n%s", h.screen())
	}
}

func TestKelpSwaysFromItsTipWhileItsBaseStaysPut(t *testing.T) {
	h := newHarness(t, named("Bubbles"))
	h.resize(120, 30)

	// at lists where kelp of colour fg is drawn, row by row.
	at := func(fg string) string {
		var where []string
		for _, p := range h.plants() {
			if p.fg == fg && strings.Contains(kelpGlyphs, p.glyph) {
				where = append(where, strconv.Itoa(p.col)+","+strconv.Itoa(p.row))
			}
		}
		return strings.Join(where, " ")
	}
	bottom := floorRow(h.view()) - 1
	var stalkFg, tipFg string
	for _, p := range h.plants() {
		if strings.Contains(kelpGlyphs, p.glyph) {
			if p.row == bottom {
				stalkFg = p.fg
			} else if p.fg != stalkFg && brightness(p.fg) > brightness(tipFg) {
				tipFg = p.fg
			}
		}
	}
	bases := func() string {
		var where []string
		for _, p := range h.plants() {
			if p.row == bottom && strings.Contains(kelpGlyphs, p.glyph) {
				where = append(where, strconv.Itoa(p.col))
			}
		}
		return strings.Join(where, " ")
	}
	firstBases, firstTips := bases(), at(tipFg)
	if firstBases == "" || firstTips == "" {
		t.Fatalf("no kelp bases or tips to watch:\n%s", h.screen())
	}

	tipMoved := false
	for range int(4 * time.Second / app.FrameInterval) {
		h.advance(app.FrameInterval)
		if got := bases(); got != firstBases {
			t.Fatalf("kelp bases moved from columns %s to %s:\n%s", firstBases, got, h.screen())
		}
		tipMoved = tipMoved || at(tipFg) != firstTips
	}
	if !tipMoved {
		t.Errorf("kelp tips stayed at %s for 4 seconds, want them to sway:\n%s", firstTips, h.screen())
	}
}

func TestFoodRestsOnTopOfKelpAndGrass(t *testing.T) {
	for _, kind := range []string{"kelp", "grass"} {
		t.Run(kind, func(t *testing.T) {
			h := newHarness(t, named("Bubbles")) // Full, so it leaves the Food be
			h.resize(120, 16)
			bottom := floorRow(h.view()) - 1
			glyphs := kelpGlyphs
			if kind == "grass" {
				glyphs = grassGlyphs
			}
			col := -1
			for _, p := range h.plants() {
				if p.row == bottom && strings.Contains(glyphs, p.glyph) {
					col = p.col
					break
				}
			}
			if col < 0 {
				t.Fatalf("no %s in the Tank:\n%s", kind, h.screen())
			}
			// The stalk's top: its highest cell, wherever it sways.
			top := bottom
			for _, p := range h.plants() {
				if p.col >= col-2 && p.col <= col+2 {
					top = min(top, p.row)
				}
			}

			h.click(col, 0)
			h.advance(15 * time.Second) // time to sink to the stalk
			seen := false
			for range int(3 * time.Second / app.FrameInterval) {
				h.advance(app.FrameInterval)
				for row := 0; row <= bottom; row++ {
					if h.cellAt(col, row) != pellet {
						continue
					}
					if row != top-1 {
						t.Fatalf("the pellet dropped on %s is at row %d, want it resting on top at row %d:\n%s", kind, row, top-1, h.screen())
					}
					seen = true
				}
			}
			if !seen {
				t.Errorf("the pellet dropped on %s never showed:\n%s", kind, h.screen())
			}
		})
	}
}
