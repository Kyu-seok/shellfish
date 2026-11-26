package app

import "time"

// Tuning numbers live here, in one place, so they are easy to adjust.

// FrameInterval is the time between animation ticks (about 20 fps).
const FrameInterval = 50 * time.Millisecond

// MinTankWidth and MinTankHeight are the smallest window, in cells, that the
// Tank is drawn in. Below either, the app asks for a bigger tank instead.
const (
	MinTankWidth  = 30
	MinTankHeight = 10
)

// SaveInterval is how often the Fish is saved while the Tank is open.
const SaveInterval = time.Minute

// GreetAfter is how long the user must be away for the Tank to greet them
// when they come back. An absence of exactly this long gets no greeting.
const GreetAfter = time.Hour

// GreetingDuration is how long the welcome-back greeting stays up.
const GreetingDuration = 5 * time.Second

// FarewellDuration is how long the farewell lasts after the user releases
// their Fish, before the Species picker opens.
const FarewellDuration = 2 * time.Second

// FarewellSwim is how long the released Fish takes to swim out of its Tank,
// leaving the rest of the farewell to say bye to an empty Tank.
const FarewellSwim = 1500 * time.Millisecond
