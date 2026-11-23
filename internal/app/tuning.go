package app

import "time"

// Tuning numbers live here, in one place, so they are easy to adjust.

// FrameInterval is the time between animation ticks (about 20 fps).
const FrameInterval = 50 * time.Millisecond
