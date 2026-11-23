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
func (k FoodKind) String() string {
	switch k {
	case Flake:
		return "flake"
	case Worm:
		return "worm"
	default:
		return "pellet"
	}
}

// Full reports whether the Fish is Full: its Hunger is within FullMargin of
// the lowest. A Full Fish ignores Food. Hunger rises all the time, so without
// the margin a Fish would stop being Full the moment after it ate.
func (f Fish) Full() bool { return f.Hunger < Full+FullMargin }

// Eat lowers Hunger by what Food of kind k is worth, never below Full.
func (f Fish) Eat(k FoodKind) Fish {
	f.Hunger = max(f.Hunger-FoodValue[k], Full)
	return f
}
