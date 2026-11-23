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

// HungerMeterCells is how many cells wide the status line's Hunger meter is.
const HungerMeterCells = 10

// SaveInterval is how often the Fish is saved while the Tank is open.
const SaveInterval = time.Minute
