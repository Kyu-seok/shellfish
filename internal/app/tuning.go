package app

import "time"

// Tuning numbers live here, in one place, so they are easy to adjust.

// FrameInterval is the time between animation ticks (about 20 fps).
const FrameInterval = 50 * time.Millisecond

// HungerMeterCells is how many cells wide the status line's Hunger meter is.
const HungerMeterCells = 10

// SaveInterval is how often the Fish is saved while the Tank is open.
const SaveInterval = time.Minute
