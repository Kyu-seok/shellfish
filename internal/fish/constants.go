package fish

// Tuning numbers for the Fish's state live here.

// Full is the lowest Hunger: a Fish that has just eaten or just been born.
const Full = 0.0

// MaxHunger is the highest Hunger a Fish can reach.
const MaxHunger = 100.0

// Colour is a Fish's colour, as a #rrggbb hex string.
type Colour string

// Palette is the small set of pastel colours a Fish can be born with.
var Palette = []Colour{
	"#FFB3BA", // pink
	"#FFDFBA", // peach
	"#FFFFBA", // lemon
	"#BAFFC9", // mint
	"#BAE1FF", // sky
	"#D7BAFF", // lilac
}

// HungerPerDay is how much Hunger rises in 24 hours, open or closed.
const HungerPerDay = 100.0

// HungryAt and StarvingAt are the lowest Hunger of the Hungry and Starving
// Mood bands. Below HungryAt the Fish is Content.
const (
	HungryAt   = 40.0
	StarvingAt = 80.0
)

// FullMargin is how far above the lowest Hunger the Fish still counts as
// Full. Hunger takes about a quarter of an hour to rise past it.
const FullMargin = 1.0

// FoodValue is how much eating each Food Kind lowers Hunger.
var FoodValue = map[FoodKind]float64{
	Pellet: 10,
	Flake:  25,
	Worm:   50,
}
