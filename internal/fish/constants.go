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
