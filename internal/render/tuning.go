package render

import "github.com/Kyu-seok/shellfish/internal/tank"

// Tuning numbers for drawing live here.

// StatusRows is how many rows below the Tank the status line takes.
const StatusRows = 1

// HungerMeterCells is how many cells wide the status line's Hunger meter is.
const HungerMeterCells = 10

// GreetingRow is the row of the Tank the welcome-back greeting starts on:
// the row below the top row of water, so it never hides Food just dropped
// there.
const GreetingRow = tank.SurfaceRows + 1

// TailSwing is how far, in half-rows, the tip of the Fish's tail swings up
// and down as it swishes.
const TailSwing = 0.9

// MouthRadius is how big the Fish's open mouth is, in cells.
const MouthRadius = 0.8

// KelpLean is how far, in columns, a kelp stalk's tip leans at the height of
// its sway, before it is held to one column either way.
const KelpLean = 1.4

// GrassBend is how far into its sway a tuft of grass must be before its
// blade leans over.
const GrassBend = 0.35

// EyeRing is how thick, in cells, the ring of body round a monochrome
// Fish's eye is: just over a pixel either way, so the eye is always a hole
// closed all round.
const EyeRing = 0.7
