// Package fish holds the Fish's state and the rules about it. It is pure
// logic: no I/O, no clock and no global randomness.
package fish

import (
	"errors"
	"math/rand/v2"
	"strings"
	"time"
)

// Fish is the one pet a user owns.
type Fish struct {
	Name   string
	Colour Colour
	Born   time.Time

	// Hunger runs from Full (0) to MaxHunger.
	Hunger float64
	// HungerUpdated is when Hunger was last brought up to date.
	HungerUpdated time.Time
	// LastSeen is when the user last had the Tank open.
	LastSeen time.Time
}

// ErrBlankName is returned for a name that is empty or only whitespace.
var ErrBlankName = errors.New("your Fish needs a name")

// Hatch names a new Fish at now. It is born Full, with a colour picked at
// random from the Palette.
func Hatch(name string, now time.Time, r *rand.Rand) (Fish, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Fish{}, ErrBlankName
	}
	return Fish{
		Name:          name,
		Colour:        Palette[r.IntN(len(Palette))],
		Born:          now,
		Hunger:        Full,
		HungerUpdated: now,
		LastSeen:      now,
	}, nil
}

// Age is the number of whole days since the Fish was named. It is never
// negative, even if the clock has moved backwards.
func (f Fish) Age(now time.Time) int {
	if !now.After(f.Born) {
		return 0
	}
	return int(now.Sub(f.Born) / (24 * time.Hour))
}
