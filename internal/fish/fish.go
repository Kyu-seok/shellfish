// Package fish holds the Fish's state and the rules about it. It is pure
// logic: no I/O, no clock and no global randomness.
package fish

import (
	"errors"
	"strings"
	"time"
)

// Fish is the one pet a user owns.
type Fish struct {
	Name string
	// Species is what kind of fish it is, for life. A Fish saved before
	// Species existed has NoSpecies until the user picks one.
	Species Species
	Born    time.Time

	// Hunger runs from MinHunger to MaxHunger.
	Hunger float64
	// HungerUpdated is when Hunger was last brought up to date.
	HungerUpdated time.Time
	// LastSeen is when the user last had the Tank open.
	LastSeen time.Time
}

// ErrBlankName is returned for a name that is empty or only whitespace.
var ErrBlankName = errors.New("your Fish needs a name")

// Hatch names a new Fish of Species s at now. It is born Full.
func Hatch(name string, s Species, now time.Time) (Fish, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Fish{}, ErrBlankName
	}
	return Fish{
		Name:          name,
		Species:       s,
		Born:          now,
		Hunger:        MinHunger,
		HungerUpdated: now,
		LastSeen:      now,
	}, nil
}

// Age is the number of whole Days since the Fish was named. It is never
// negative, even if the clock has moved backwards.
func (f Fish) Age(now time.Time) int {
	if !now.After(f.Born) {
		return 0
	}
	return int(now.Sub(f.Born) / Day)
}
