package tank

import (
	"math"
	"math/rand/v2"
	"slices"

	"github.com/Kyu-seok/shellfish/internal/fish"
)

// Bubble is one bubble rising through the Tank.
type Bubble struct {
	Col, Row int
	// Size grows from 0 (small) to 2 (large) as the bubble rises.
	Size int
}

// FlourishKind is what a Flourish shows.
type FlourishKind int

const (
	Heart FlourishKind = iota
	Note
)

// Flourish is a heart or note floating up from the Fish.
type Flourish struct {
	Col, Row int
	Kind     FlourishKind
	// Fading is set near the end of its life.
	Fading bool
}

type bubble struct {
	x, y, from float64 // from is the row it started rising at
}

type flourish struct {
	x, y float64
	kind FlourishKind
	age  float64
}

// decor is everything in the Tank that is only for show: blinking, bubbles,
// hearts and notes. It has its own random source, so it never changes how
// the Fish swims.
type decor struct {
	rand       *rand.Rand
	blinkIn    float64 // seconds until the next blink starts
	blinking   float64 // seconds left with the eyes shut
	ambientIn  float64 // seconds until the next bubble rises from the floor
	puffIn     float64 // seconds until a Content Fish blows a bubble
	noteIn     float64 // seconds until a Content Fish shows a heart or note
	bubbles    []bubble
	flourishes []flourish
}

func newDecor(r *rand.Rand) decor {
	d := decor{rand: r}
	d.blinkIn = d.between(BlinkMin, BlinkMax)
	d.ambientIn = d.between(AmbientMin, AmbientMax)
	d.puffIn = d.between(PuffMin, PuffMax)
	d.noteIn = d.between(NoteMin, NoteMax)
	return d
}

func (d *decor) between(lo, hi float64) float64 { return between(d.rand, lo, hi) }

// decorate advances the decorations by s seconds, for a Fish in mood.
func (t *Tank) decorate(s float64, mood fish.Mood) {
	d := &t.decor
	f := &t.fish

	d.blinking = max(d.blinking-s, 0)
	if d.blinkIn -= s; d.blinkIn <= 0 {
		d.blinking, d.blinkIn = BlinkTime, d.between(BlinkMin, BlinkMax)
	}

	for i := range d.bubbles {
		d.bubbles[i].y -= BubbleRise * s
	}
	d.bubbles = slices.DeleteFunc(d.bubbles, func(b bubble) bool { return b.y < -0.5 })
	for i := range d.flourishes {
		// It floats up as far as the top of the Tank, and fades there.
		d.flourishes[i].y = max(d.flourishes[i].y-FlourishRise*s, 0)
		d.flourishes[i].age += s
	}
	d.flourishes = slices.DeleteFunc(d.flourishes, func(fl flourish) bool { return fl.age >= FlourishLife })

	if d.ambientIn -= s; d.ambientIn <= 0 {
		d.ambientIn = d.between(AmbientMin, AmbientMax)
		if t.width > 0 && t.waterRows() > 0 {
			y := float64(t.waterRows() - 1)
			d.bubbles = append(d.bubbles, bubble{x: float64(d.rand.IntN(t.width)), y: y, from: y})
		}
	}

	if mood != fish.Content {
		return // only a Content Fish blows bubbles and shows hearts and notes
	}
	row := t.fishRow()
	if d.puffIn -= s; d.puffIn <= 0 {
		d.puffIn = d.between(PuffMin, PuffMax)
		// Just in front of its mouth.
		x := f.x + FishWidth - 1
		if f.facingLeft {
			x = f.x - 1
		}
		if !f.mouthOpen && x >= 0 && x < float64(t.width) {
			y := row + 1
			d.bubbles = append(d.bubbles, bubble{x: x, y: y, from: y})
		}
	}
	if d.noteIn -= s; d.noteIn <= 0 {
		d.noteIn = d.between(NoteMin, NoteMax)
		kind := Heart
		if d.rand.IntN(2) == 0 {
			kind = Note
		}
		t.flourish(f.x+math.Floor(FishWidth/2), row-1, kind)
	}
}

// flourish starts a heart or note floating up from col, row.
func (t *Tank) flourish(col, row float64, kind FlourishKind) {
	row = max(row, 0)
	t.decor.flourishes = append(t.decor.flourishes, flourish{x: col, y: row, kind: kind})
}

// ate celebrates a meal at col, row: a heart floats up from where the Fish
// ate, and it opens its eyes wide rather than blink.
func (t *Tank) ate(m Meal) {
	t.decor.blinking = 0
	t.flourish(float64(m.Col), float64(m.Row-1), Heart)
}

// fishRow is the row the top of the Fish is drawn at, bob included.
func (t *Tank) fishRow() float64 {
	return math.Round(t.fish.y + BobAmplitude*math.Sin(t.fish.bob))
}

// trimDecor drops decorations that a resize left outside the water.
func (t *Tank) trimDecor() {
	d := &t.decor
	outside := func(x, y float64) bool { return x >= float64(t.width) || y > float64(t.waterRows()-1) }
	d.bubbles = slices.DeleteFunc(d.bubbles, func(b bubble) bool { return outside(b.x, b.y) })
	d.flourishes = slices.DeleteFunc(d.flourishes, func(fl flourish) bool { return outside(fl.x, fl.y) })
}

// snapshotDecor adds the decorations to snap.
func (t *Tank) snapshotDecor(snap *Snapshot) {
	d := t.decor
	snap.Fish.Blinking = d.blinking > 0
	for _, b := range d.bubbles {
		size := 0
		switch risen := b.from - b.y; {
		case risen >= BubbleLarge:
			size = 2
		case risen >= BubbleMedium:
			size = 1
		}
		snap.Bubbles = append(snap.Bubbles, Bubble{Col: int(b.x), Row: int(math.Round(b.y)), Size: size})
	}
	for _, fl := range d.flourishes {
		snap.Flourishes = append(snap.Flourishes, Flourish{
			Col:    int(fl.x),
			Row:    int(math.Round(fl.y)),
			Kind:   fl.kind,
			Fading: fl.age >= FlourishLife-FlourishFade,
		})
	}
}
