package render

import (
	"strconv"
	"strings"
)

// Detail is how finely the Fish is drawn (ADR 0004). It covers the same
// cells either way.
type Detail int

const (
	// Fine draws the Fish in sextants, 2×3 pixels a cell, which many fonts
	// lack.
	Fine Detail = iota
	// Coarse draws the Fish in quadrants, 2×2 pixels a cell, which nearly
	// every font has.
	Coarse
)

// pixels is how many columns and rows of pixels a cell is split into, and
// glyph the character for a cell with the pixels in mask filled.
func (d Detail) pixels() (cols, rows int) {
	if d == Coarse {
		return 2, 2
	}
	return 2, 3
}

func (d Detail) glyph(mask int) rune {
	if d == Coarse {
		return quadrants[mask]
	}
	return sextant(mask)
}

// Other is the Detail d switches to.
func (d Detail) Other() Detail {
	if d == Coarse {
		return Fine
	}
	return Coarse
}

// quadrants is the character for each 2×2 mask, bits running left to right,
// top to bottom: top-left 1, top-right 2, bottom-left 4, bottom-right 8.
var quadrants = [16]rune{' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛', '▗', '▚', '▐', '▜', '▄', '▙', '▟', '█'}

// GuessDetail guesses which Detail the terminal whose environment variables
// getenv reads can show, and names the terminal, so a Detail the user picks
// can be remembered for it. Only a terminal known to draw sextants itself,
// whatever the font, at a version that does, gets Fine: a wrong Coarse costs
// a little detail, but a wrong Fine shows boxes. A multiplexer hides which
// terminal is outside it, so tmux and any TERM_PROGRAM not in terminals get
// Coarse, even with another terminal's variables left over from outside.
func GuessDetail(getenv func(string) string) (terminal string, d Detail) {
	if getenv("TMUX") != "" {
		return "tmux", Coarse
	}
	program := getenv("TERM_PROGRAM")
	for _, t := range terminals {
		if program != "" && t.program == program || program == "" && t.is != nil && t.is(getenv) {
			if t.fine(getenv) {
				return t.name, Fine
			}
			return t.name, Coarse
		}
	}
	switch {
	case program != "":
		return program, Coarse
	case getenv("TERM") != "":
		return getenv("TERM"), Coarse
	}
	return "unknown", Coarse
}

// terminal is what GuessDetail knows of one terminal: its name, how to tell
// it is the one running, by its TERM_PROGRAM or else by is, and whether it
// draws sextants itself.
type terminal struct {
	name    string
	program string
	is      func(getenv func(string) string) bool
	fine    func(getenv func(string) string) bool
}

// terminals is every terminal GuessDetail recognises, with the versions that
// draw sextants themselves, whatever the font.
var terminals = []terminal{
	{name: "ghostty", program: "ghostty", fine: always},
	{name: "WezTerm", program: "WezTerm", fine: always},
	{name: "rio", program: "rio", fine: always},
	{name: "contour", program: "contour", fine: always},
	{name: "iTerm.app", program: "iTerm.app", fine: since("TERM_PROGRAM_VERSION", 3, 6, 7)},
	{name: "vscode", program: "vscode", fine: since("TERM_PROGRAM_VERSION", 1, 108)},
	{name: "kitty", is: either(set("KITTY_WINDOW_ID"), termIs("xterm-kitty")), fine: always},
	{name: "alacritty", is: either(set("ALACRITTY_WINDOW_ID"), termIs("alacritty")), fine: always},
	{name: "foot", is: either(termIs("foot"), termIs("foot-extra")), fine: always},
	{name: "konsole", is: set("KONSOLE_VERSION"), fine: since("KONSOLE_VERSION", 220400)},
	{name: "vte", is: set("VTE_VERSION"), fine: since("VTE_VERSION", 6000)},
	// Windows Terminal draws sextants from 1.25, but doesn't say which
	// version is running.
	{name: "windows-terminal", is: set("WT_SESSION"), fine: never},
}

func always(func(string) string) bool { return true }

func never(func(string) string) bool { return false }

func set(name string) func(func(string) string) bool {
	return func(getenv func(string) string) bool { return getenv(name) != "" }
}

func termIs(term string) func(func(string) string) bool {
	return func(getenv func(string) string) bool { return getenv("TERM") == term }
}

func either(a, b func(func(string) string) bool) func(func(string) string) bool {
	return func(getenv func(string) string) bool { return a(getenv) || b(getenv) }
}

// since reports whether the version in the variable name is first or later.
// A version it can't read is not.
func since(name string, first ...int) func(func(string) string) bool {
	return func(getenv func(string) string) bool {
		v, ok := version(getenv(name))
		if !ok {
			return false
		}
		for i, f := range first {
			n := 0
			if i < len(v) {
				n = v[i]
			}
			if n != f {
				return n > f
			}
		}
		return true
	}
}

// version reads a version such as 3.6.7 or 3.7.0beta2 as its numbers, each
// part's leading digits. It fails if the first part has none.
func version(s string) ([]int, bool) {
	var v []int
	for _, part := range strings.Split(s, ".") {
		end := 0
		for end < len(part) && part[end] >= '0' && part[end] <= '9' {
			end++
		}
		if end == 0 {
			break
		}
		n, err := strconv.Atoi(part[:end])
		if err != nil {
			return nil, false
		}
		v = append(v, n)
		if end < len(part) {
			break // the rest is a pre-release tag
		}
	}
	return v, len(v) > 0
}

// String is the Detail's name, as it is remembered: fine or coarse.
func (d Detail) String() string {
	if d == Coarse {
		return "coarse"
	}
	return "fine"
}

// ParseDetail is the Detail named s, as String names it, or false if s
// names none.
func ParseDetail(s string) (Detail, bool) {
	switch s {
	case "fine":
		return Fine, true
	case "coarse":
		return Coarse, true
	}
	return Fine, false
}
