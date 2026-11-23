package fish

import "time"

// Mood is how the Fish feels, decided entirely by its Hunger.
type Mood int

const (
	Content Mood = iota
	Hungry
	Starving
)

// Mood is the Fish's Mood at its current Hunger.
func (f Fish) Mood() Mood {
	switch {
	case f.Hunger >= StarvingAt:
		return Starving
	case f.Hunger >= HungryAt:
		return Hungry
	default:
		return Content
	}
}

// CatchUp raises Hunger for the time since HungerUpdated, at HungerPerDay,
// capped at MaxHunger, and marks Hunger as up to date at now. If the clock has
// moved backwards, no time has passed.
func (f Fish) CatchUp(now time.Time) Fish {
	if elapsed := now.Sub(f.HungerUpdated); elapsed > 0 {
		// Multiply before dividing so whole-hour amounts come out exact.
		f.Hunger = min(f.Hunger+float64(elapsed)*HungerPerDay/float64(24*time.Hour), MaxHunger)
	}
	f.HungerUpdated = now
	return f
}
