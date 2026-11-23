package fish

// FoodKind is one type of Food, such as a pellet, flake or worm.
type FoodKind int

const (
	Pellet FoodKind = iota
	Flake
	Worm
)

// FoodKinds are all the Food Kinds, in the order the user picks them with
// the keys 1, 2 and 3.
var FoodKinds = []FoodKind{Pellet, Flake, Worm}

// String is the Food Kind's name.
func (k FoodKind) String() string { return foodKinds[k].name }

// Glyph is how Food of this kind is drawn.
func (k FoodKind) Glyph() rune { return foodKinds[k].glyph }

// Value is how much eating Food of this kind lowers Hunger.
func (k FoodKind) Value() float64 { return foodKinds[k].value }

// SinkSpeed is how fast Food of this kind sinks, in rows per second.
func (k FoodKind) SinkSpeed() float64 { return foodKinds[k].sink }

// Colour is the colour Food of this kind is drawn in, as a #rrggbb hex string.
func (k FoodKind) Colour() string { return foodKinds[k].colour }

// Full reports whether the Fish is Full: its Hunger is within FullMargin of
// the lowest. A Full Fish ignores Food. Hunger rises all the time, so without
// the margin a Fish would stop being Full the moment after it ate.
func (f Fish) Full() bool { return f.Hunger < MinHunger+FullMargin }

// Eat lowers Hunger by what Food of kind k is worth, never below MinHunger.
func (f Fish) Eat(k FoodKind) Fish {
	f.Hunger = max(f.Hunger-k.Value(), MinHunger)
	return f
}
