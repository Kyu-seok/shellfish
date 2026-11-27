// Command shots draws the README's pictures of shellfish: it runs the real app
// with a scripted clock, input and random seed, and draws each frame straight
// to pixels. The Fish's sextant cells become exact 2×3 blocks of colour, which
// no screen recorder can promise, since few fonts have sextants; other
// characters come from a font, drawn without anti-aliasing so a frame holds
// only a few exact colours.
//
// Run it from the repo root (make shots): it writes docs/*.gif and docs/*.png.
// It reads system fonts, so it runs on macOS as it is; elsewhere pass -fonts.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"github.com/Kyu-seok/shellfish/internal/app"
	"github.com/Kyu-seok/shellfish/internal/fish"
)

// The size of one terminal cell in pixels, and the font drawn in it.
const (
	cellW, cellH = 12, 24
	fontSize     = 19
	pad          = 16 // pixels of background around each picture
	gifStep      = 2  // app frames per GIF frame: 20 fps → 10 fps
	// tileW and tileH are the window, in cells, each small Tank in the
	// Species and Mood pictures is drawn in.
	tileW, tileH = 36, 13
)

var (
	background = color.RGBA{0x10, 0x14, 0x1a, 0xff}
	foreground = color.RGBA{0xd8, 0xde, 0xe9, 0xff}
	// start is when every scripted run begins, so the pictures never change.
	start = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

func main() {
	out := flag.String("out", "docs", "where to write the pictures")
	fonts := flag.String("fonts", strings.Join([]string{
		"/System/Library/Fonts/Menlo.ttc",
		"/System/Library/Fonts/Apple Symbols.ttf",
		filepath.Join(os.Getenv("HOME"), "Library/Fonts/HackNerdFontMono-Regular.ttf"),
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
	}, ","), "comma-separated font files, tried in order for each character")
	flag.Parse()
	lipgloss.SetColorProfile(termenv.TrueColor)
	if err := run(*out, strings.Split(*fonts, ",")); err != nil {
		fmt.Fprintln(os.Stderr, "shots:", err)
		os.Exit(1)
	}
}

func run(out string, fontFiles []string) error {
	faces, err := loadFaces(fontFiles)
	if err != nil {
		return err
	}
	p := painter{faces: faces, missing: map[rune]bool{}}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	steps := []func() error{
		func() error { return p.hero(filepath.Join(out, "hero.gif")) },
		func() error { return p.species(filepath.Join(out, "species.png")) },
		func() error { return p.moods(filepath.Join(out, "moods.png")) },
		func() error { return p.picker(filepath.Join(out, "picker.png")) },
	}
	for _, s := range steps {
		if err := s(); err != nil {
			return err
		}
	}
	if len(p.missing) > 0 {
		var rs []string
		for r := range p.missing {
			rs = append(rs, fmt.Sprintf("%q", r))
		}
		return fmt.Errorf("no font has %s: pass one that does with -fonts", strings.Join(rs, " "))
	}
	return nil
}

// --- Driving the app --------------------------------------------------------

type clock struct{ now time.Time }

func (c *clock) Now() time.Time             { return c.now }
func (c *clock) Tick(time.Duration) tea.Cmd { return nil }

// session is one run of the app, driven as a terminal would drive it.
type session struct {
	clock *clock
	model tea.Model
	dir   string
	seed  uint64
}

// launch opens the app on dir at the clock's time.
func launch(dir string, c *clock, seed uint64) *session {
	m := app.New(app.Deps{Clock: c, Rand: rand.New(rand.NewPCG(seed, 2)), DataDir: dir})
	return &session{clock: c, model: m, dir: dir, seed: seed}
}

// newFish starts on an empty data folder, picks sp and names the Fish, then
// closes the Tank and opens it again after leaving the Fish unfed for long
// enough that its Hunger is hunger.
func newFish(sp fish.Species, hunger float64, seed uint64, width, height int) (*session, error) {
	dir, err := os.MkdirTemp("", "shellfish-shots-")
	if err != nil {
		return nil, err
	}
	c := &clock{now: start}
	s := launch(dir, c, seed)
	s.resize(width, height)
	for range indexOf(sp) {
		s.send(tea.KeyMsg{Type: tea.KeyRight})
	}
	s.send(tea.KeyMsg{Type: tea.KeyEnter})
	for _, r := range "Bubbles" {
		s.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	s.send(tea.KeyMsg{Type: tea.KeyEnter})
	s.close()
	c.now = c.now.Add(time.Duration(hunger / fish.HungerPerDay * float64(fish.Day)))
	s = launch(dir, c, seed)
	s.resize(width, height)
	s.advance(app.GreetingDuration + time.Second) // past the welcome-back greeting
	return s, nil
}

func indexOf(sp fish.Species) int {
	for i, s := range fish.AllSpecies {
		if s == sp {
			return i
		}
	}
	return 0
}

func (s *session) send(msgs ...tea.Msg) {
	for _, m := range msgs {
		s.model, _ = s.model.Update(m)
	}
}

func (s *session) resize(w, h int) { s.send(tea.WindowSizeMsg{Width: w, Height: h}) }

func (s *session) click(col, row int) {
	s.send(
		tea.MouseMsg{X: col, Y: row, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		tea.MouseMsg{X: col, Y: row, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft},
	)
}

// tick moves time on by one animation frame.
func (s *session) tick() {
	s.clock.now = s.clock.now.Add(app.FrameInterval)
	s.send(app.TickMsg{Time: s.clock.now})
}

func (s *session) advance(d time.Duration) {
	for range int(d / app.FrameInterval) {
		s.tick()
	}
}

// close quits the app, which saves the Fish, and lets go of its lock.
func (s *session) close() {
	s.send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	s.model.(app.Model).Close()
}

func (s *session) screen() grid { return parse(s.model.View()) }

// --- The terminal screen ----------------------------------------------------

type cell struct {
	r      rune
	fg, bg color.RGBA
	hasBG  bool
	faint  bool
}

type grid [][]cell

// parse reads the app's view, true-colour SGR codes and all, into cells.
func parse(view string) grid {
	var g grid
	for _, line := range strings.Split(view, "\n") {
		var row []cell
		cur := cell{fg: foreground}
		for len(line) > 0 {
			if strings.HasPrefix(line, "\x1b[") {
				end := strings.IndexByte(line, 'm')
				if end < 0 {
					break
				}
				cur = sgr(cur, line[2:end])
				line = line[end+1:]
				continue
			}
			r, size := decodeRune(line)
			line = line[size:]
			c := cur
			c.r = r
			row = append(row, c)
			for range ansi.StringWidth(string(r)) - 1 {
				row = append(row, cell{r: 0, bg: c.bg, hasBG: c.hasBG})
			}
		}
		g = append(g, row)
	}
	return g
}

func decodeRune(s string) (rune, int) {
	for i, r := range s {
		if i == 0 {
			return r, len(string(r))
		}
	}
	return 0, 1
}

// sgr applies one Select Graphic Rendition sequence's parameters to c.
func sgr(c cell, params string) cell {
	ps := strings.Split(params, ";")
	num := func(i int) uint8 {
		if i >= len(ps) {
			return 0
		}
		n, _ := strconv.Atoi(ps[i])
		return uint8(n)
	}
	for i := 0; i < len(ps); i++ {
		switch ps[i] {
		case "", "0":
			c = cell{fg: foreground}
		case "2":
			c.faint = true
		case "22":
			c.faint = false
		case "39":
			c.fg = foreground
		case "49":
			c.hasBG = false
		case "38", "48":
			if i+4 < len(ps) && ps[i+1] == "2" {
				col := color.RGBA{num(i + 2), num(i + 3), num(i + 4), 0xff}
				if ps[i] == "38" {
					c.fg = col
				} else {
					c.bg, c.hasBG = col, true
				}
				i += 4
			}
		}
	}
	return c
}

// --- Drawing ----------------------------------------------------------------

type painter struct {
	faces   []fontFace
	missing map[rune]bool
}

type fontFace struct {
	font *opentype.Font
	face font.Face
}

func loadFaces(files []string) ([]fontFace, error) {
	var faces []fontFace
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // a missing fallback is fine; a missing glyph is reported
		}
		coll, err := opentype.ParseCollection(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		f, err := coll.Font(0)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		faces = append(faces, fontFace{f, face})
	}
	if len(faces) == 0 {
		return nil, fmt.Errorf("none of the fonts could be read: %s", strings.Join(files, ", "))
	}
	return faces, nil
}

// draw paints g into a picture with pad pixels of background around it.
func (p *painter) draw(g grid) *image.RGBA {
	w := 0
	for _, row := range g {
		w = max(w, len(row))
	}
	img := image.NewRGBA(image.Rect(0, 0, w*cellW+2*pad, len(g)*cellH+2*pad))
	draw.Draw(img, img.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
	for y, row := range g {
		for x, c := range row {
			p.cell(img, pad+x*cellW, pad+y*cellH, c)
		}
	}
	return img
}

func (p *painter) cell(img *image.RGBA, x0, y0 int, c cell) {
	box := image.Rect(x0, y0, x0+cellW, y0+cellH)
	if c.hasBG {
		draw.Draw(img, box, image.NewUniform(c.bg), image.Point{}, draw.Src)
	}
	fg := c.fg
	if c.faint {
		fg = blend(fg, background, 0.5)
	}
	if mask, ok := blocks(c.r); ok {
		// 2×3 pixels, bit 1 top-left, 2 top-right, 4 middle-left and so on.
		for bit := range 6 {
			if mask&(1<<bit) == 0 {
				continue
			}
			px, py := bit%2, bit/2
			sub := image.Rect(x0+px*cellW/2, y0+py*cellH/3, x0+(px+1)*cellW/2, y0+(py+1)*cellH/3)
			draw.Draw(img, sub, image.NewUniform(fg), image.Point{}, draw.Src)
		}
		return
	}
	if c.r == ' ' || c.r == 0 {
		return
	}
	p.glyph(img, box, c.r, fg)
}

// blocks is the 2×3 pixel mask a sextant or half or full block fills.
func blocks(r rune) (int, bool) {
	switch {
	case r == '▌':
		return 21, true
	case r == '▐':
		return 42, true
	case r == '█':
		return 63, true
	case r >= 0x1FB00 && r <= 0x1FB3B:
		m := int(r-0x1FB00) + 1
		if m >= 21 {
			m++
		}
		if m >= 42 {
			m++
		}
		return m, true
	}
	return 0, false
}

// glyph draws r in the first font that has it, without anti-aliasing.
func (p *painter) glyph(img *image.RGBA, box image.Rectangle, r rune, fg color.RGBA) {
	var buf sfnt.Buffer
	for _, f := range p.faces {
		if i, err := f.font.GlyphIndex(&buf, r); err != nil || i == 0 {
			continue
		}
		alpha := image.NewAlpha(image.Rect(0, 0, cellW*2, cellH))
		d := font.Drawer{Dst: alpha, Src: image.Opaque, Face: f.face}
		adv, _ := f.face.GlyphAdvance(r)
		x := (fixed.I(cellW) - adv) / 2 // centred, for fonts a little wider or narrower
		m := f.face.Metrics()
		y := (fixed.I(cellH)-(m.Ascent+m.Descent))/2 + m.Ascent
		d.Dot = fixed.Point26_6{X: x, Y: y}
		d.DrawString(string(r))
		for py := range cellH {
			for px := range cellW {
				if alpha.AlphaAt(px, py).A >= 0x80 {
					img.SetRGBA(box.Min.X+px, box.Min.Y+py, fg)
				}
			}
		}
		return
	}
	p.missing[r] = true
}

func blend(a, b color.RGBA, k float64) color.RGBA {
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*k + float64(y)*(1-k)) }
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 0xff}
}

// label is a picture of text alone, drawn as the Tank draws characters.
func (p *painter) label(text string, width int) grid {
	row := make([]cell, width)
	for i := range row {
		row[i] = cell{r: ' ', fg: foreground}
	}
	start := max((width-len([]rune(text)))/2, 0)
	for i, r := range []rune(text) {
		if start+i < width {
			row[start+i].r = r
		}
	}
	return grid{row}
}

// stack joins grids into one picture: side by side in rows of perRow, each
// with its label underneath, one empty column between them.
func stack(tiles []grid, perRow int) grid {
	var out grid
	for i := 0; i < len(tiles); i += perRow {
		group := tiles[i:min(i+perRow, len(tiles))]
		height := 0
		for _, t := range group {
			height = max(height, len(t))
		}
		for y := range height {
			var row []cell
			for j, t := range group {
				if j > 0 {
					row = append(row, cell{r: ' ', fg: foreground}, cell{r: ' ', fg: foreground})
				}
				if y < len(t) {
					row = append(row, t[y]...)
				}
			}
			out = append(out, row)
		}
		if i+perRow < len(tiles) {
			out = append(out, nil)
		}
	}
	return out
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	return f.Close()
}

// writeGIF encodes frames, storing for each only the box that changed since
// the one before, with a palette of exactly the colours in that box.
func writeGIF(path string, frames []*image.RGBA, delay int) error {
	out := &gif.GIF{}
	var prev *image.RGBA
	for _, f := range frames {
		box := f.Bounds()
		if prev != nil {
			box = changed(prev, f)
			if box.Empty() {
				out.Delay[len(out.Delay)-1] += delay
				continue
			}
		}
		pal, err := paletteOf(f, box)
		if err != nil {
			return err
		}
		pi := image.NewPaletted(box, pal)
		draw.Draw(pi, box, f, box.Min, draw.Src)
		out.Image = append(out.Image, pi)
		out.Delay = append(out.Delay, delay)
		out.Disposal = append(out.Disposal, gif.DisposalNone)
		prev = f
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := gif.EncodeAll(file, out); err != nil {
		return err
	}
	fmt.Println("wrote", path)
	return file.Close()
}

func changed(a, b *image.RGBA) image.Rectangle {
	var box image.Rectangle
	r := a.Bounds()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

func paletteOf(img *image.RGBA, box image.Rectangle) (color.Palette, error) {
	seen := map[color.RGBA]bool{}
	var pal color.Palette
	for y := box.Min.Y; y < box.Max.Y; y++ {
		for x := box.Min.X; x < box.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if !seen[c] {
				seen[c] = true
				pal = append(pal, c)
			}
		}
	}
	if len(pal) > 256 {
		return nil, fmt.Errorf("a frame has %d colours, more than a GIF holds", len(pal))
	}
	return pal, nil
}

// fishCell is the cell nearest the middle of the Fish, where a click pats it.
func fishCell(g grid) (col, row int, ok bool) {
	var xs, ys []int
	for y, r := range g {
		for x, c := range r {
			if _, isBlock := blocks(c.r); isBlock {
				xs, ys = append(xs, x), append(ys, y)
			}
		}
	}
	if len(xs) == 0 {
		return 0, 0, false
	}
	sx, sy := 0, 0
	for i := range xs {
		sx, sy = sx+xs[i], sy+ys[i]
	}
	mx, my := sx/len(xs), sy/len(ys)
	best := -1
	for i := range xs {
		d := (xs[i]-mx)*(xs[i]-mx) + (ys[i]-my)*(ys[i]-my)
		if best < 0 || d < (xs[best]-mx)*(xs[best]-mx)+(ys[best]-my)*(ys[best]-my) {
			best = i
		}
	}
	return xs[best], ys[best], true
}

// --- The pictures -----------------------------------------------------------

// hero is the GIF at the top of the README: a Hungry Fish swimming, Food
// dropped in front of it, the Fish noticing it and darting over to eat it,
// then a Pat.
func (p *painter) hero(path string) error {
	s, err := newFish(fish.Clownfish, (fish.HungryAt+fish.StarvingAt)/2, 3, 72, 18)
	if err != nil {
		return err
	}
	defer s.close()
	var frames []*image.RGBA
	record := func(d time.Duration) {
		for i := range int(d / app.FrameInterval) {
			s.tick()
			if i%gifStep == 0 {
				frames = append(frames, p.draw(s.screen()))
			}
		}
	}
	record(1500 * time.Millisecond)
	s.click(52, 7)
	record(4 * time.Second)
	col, row, ok := fishCell(s.screen())
	if !ok {
		return fmt.Errorf("hero: no Fish on screen to pat")
	}
	s.click(col, row)
	record(3 * time.Second)
	return writeGIF(path, frames, 100/(1000/int(app.FrameInterval.Milliseconds()*gifStep)))
}

// species shows six of the Species, each in its own small Tank.
func (p *painter) species(path string) error {
	var tiles []grid
	for _, sp := range []fish.Species{fish.Clownfish, fish.Goldfish, fish.Pufferfish, fish.BlueTang, fish.Betta, fish.Koi} {
		s, err := newFish(sp, 0, 5, tileW, tileH)
		if err != nil {
			return err
		}
		s.advance(2 * time.Second)
		g := s.screen()
		s.close()
		g = g[:len(g)-1] // without the status line
		tiles = append(tiles, append(g, p.label(sp.Title(), tileW)...))
	}
	return writePNG(path, p.draw(stack(tiles, 3)))
}

// moods shows one Fish Content, Hungry and Starving.
func (p *painter) moods(path string) error {
	var tiles []grid
	// Hunger in the middle of each Mood's band.
	for _, m := range []struct {
		name   string
		hunger float64
	}{
		{"Content", fish.HungryAt / 2},
		{"Hungry", (fish.HungryAt + fish.StarvingAt) / 2},
		{"Starving", (fish.StarvingAt + fish.MaxHunger) / 2},
	} {
		s, err := newFish(fish.Clownfish, m.hunger, 7, tileW, tileH)
		if err != nil {
			return err
		}
		s.advance(10 * time.Second) // a Starving Fish droops low first
		g := s.screen()
		s.close()
		g = g[:len(g)-1]
		tiles = append(tiles, append(g, p.label(m.name, tileW)...))
	}
	return writePNG(path, p.draw(stack(tiles, 3)))
}

// picker is the Species picker a first launch opens on.
func (p *painter) picker(path string) error {
	dir, err := os.MkdirTemp("", "shellfish-shots-")
	if err != nil {
		return err
	}
	s := launch(dir, &clock{now: start}, 1)
	defer s.close()
	s.resize(64, 18)
	s.advance(time.Second)
	return writePNG(path, p.draw(s.screen()))
}
