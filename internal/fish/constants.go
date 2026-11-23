package fish

import "time"

// Tuning numbers for the Fish's state live here.

// Day is how long a day is, for Age, Hunger and how long the user was away.
const Day = 24 * time.Hour

// MinHunger is the lowest Hunger: a Fish that has just been born or has
// eaten its fill.
const MinHunger = 0.0

// MaxHunger is the highest Hunger a Fish can reach.
const MaxHunger = 100.0

// HungerPerDay is how much Hunger rises in a Day, open or closed.
const HungerPerDay = 100.0

// HungryAt and StarvingAt are the lowest Hunger of the Hungry and Starving
// Mood bands. Below HungryAt the Fish is Content.
const (
	HungryAt   = 40.0
	StarvingAt = 80.0
)

// FullMargin is how far above MinHunger the Fish still counts as Full.
// Hunger takes about a quarter of an hour to rise past it.
const FullMargin = 1.0

// foodKinds describes each Food Kind, in the order of the FoodKind
// constants: adding a kind means adding a constant and a row here.
var foodKinds = []struct {
	name   string
	glyph  rune    // how it is drawn
	value  float64 // how much eating it lowers Hunger
	sink   float64 // how fast it sinks, in rows per second
	colour string  // how it is coloured, as a #rrggbb hex string
}{
	Pellet: {name: "pellet", glyph: '•', value: 10, sink: 1.2, colour: "#A0703C"},
	Flake:  {name: "flake", glyph: '*', value: 25, sink: 0.8, colour: "#FF9F40"}, // flutters down
	Worm:   {name: "worm", glyph: 'ʃ', value: 50, sink: 1.6, colour: "#FF8FB0"},
}
