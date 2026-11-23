package tank

import "github.com/Kyu-seok/shellfish/internal/fish"

// Tuning numbers for the Tank simulation. Distances are in cells, times in
// seconds. A cell is about twice as tall as it is wide, so vertical speeds are
// kept lower than horizontal ones.

const (
	// CruiseSpeed is the Fish's top speed on an ordinary trip.
	CruiseSpeed = 8.0
	// Accel is how quickly the Fish speeds up and slows down.
	Accel = 10.0
	// VerticalScale scales speeds and accelerations for up-and-down movement.
	VerticalScale = 0.4
	// DartChance is the chance that a trip is a quick dart ahead, of DartMin
	// to DartMax columns, rising or sinking by up to DartRise rows, at up to
	// DartSpeed. DartSpeed must stay at or below 20 so the Fish never skips
	// a cell between frames.
	DartChance = 0.2
	DartMin    = 10.0
	DartMax    = 18.0
	DartRise   = 1.5
	DartSpeed  = 18.0
	DartAccel  = 60.0
	// TurnHold is how long the Fish holds still before it turns around.
	TurnHold = 0.35
	// ArriveDistance is how close to a wander target counts as arrived.
	ArriveDistance = 0.3
	// HoverMin and HoverMax bound how long the Fish pauses between trips.
	HoverMin = 2.5
	HoverMax = 5.0
	// BobAmplitude and BobPeriod shape the gentle up-and-down bob. An
	// amplitude above half a cell makes the bob show wherever the Fish is, and
	// a pause outlasts a whole bob even after the Fish brakes.
	BobAmplitude = 0.6
	BobPeriod    = 1.6
	// SurfaceRows is how many rows at the top of the Tank are the water's surface.
	// The Fish, Food, bubbles, hearts and notes all stay below them.
	SurfaceRows = 1
	// RiseStep is how far, in rows, each try rises when looking for a place
	// above the seaweed where the Fish's mouth reaches Food.
	RiseStep = 0.25
	// MaxTrip gives up on a trip that takes longer than this.
	MaxTrip = 15.0
	// TargetTries is how many random wander targets to try before giving up
	// and hovering a little longer.
	TargetTries = 30

	// SeaweedGap is the fewest empty columns of floor between two stalks, or
	// between a stalk and a wall, unless the stalk stands against the wall. It
	// leaves the Fish room to reach Food resting anywhere on the floor.
	SeaweedGap = 18
	// SeaweedMinHeight and SeaweedMaxHeight bound a stalk's height in rows.
	SeaweedMinHeight = 2
	SeaweedMaxHeight = 5
	// SeaweedSwayPeriod is how long a stalk takes to sway one way and back.
	SeaweedSwayPeriod = 1.6
	// ShellClearance is the fewest columns of sand between a shell and a stalk,
	// so a swaying kelp tip or a grass tuft never stands on a shell.
	ShellClearance = 3
	// FishClearance is the rows kept free above the tallest stalk so the Fish
	// can always swim over it.
	FishClearance = 2

	// TailBaseRate is how many times a second the tail swishes while still.
	TailBaseRate = 1.5
	// TailRatePerSpeed adds swishes per second for each cell per second of speed.
	TailRatePerSpeed = 0.8
)

// Food.
const (
	// MaxFood is the most Food the Tank holds at once. Extra drops are ignored.
	MaxFood = 8
	// FoodRestTime is how long Food rests on the floor before it is gone, and
	// FoodFadeTime is how much of the end of that it spends fading.
	FoodRestTime = 30.0
	FoodFadeTime = 5.0
)

// The Fish going for Food.
const (
	// ChaseSpeed and ChaseAccel are the Fish's top speed and acceleration on
	// its way to Food: quicker than a cruise.
	ChaseSpeed = 12.0
	ChaseAccel = 24.0
	// ReverseSpeed is how fast the Fish can back up to line its mouth up with
	// Food just behind it, rather than swimming off and turning around.
	ReverseSpeed = 2.5
	// BackUpDistance is the furthest the Fish backs up to reach Food. Further
	// than that it turns and swims there, then turns back to face the Food.
	BackUpDistance = 3.0
	// TurnCost is how many cells of swimming one turn around is worth, when
	// the Fish weighs up which Food is nearest and which side to eat it from.
	TurnCost = 8.0
	// EatReachCols and EatReachRows are how close Food must be to the Fish's
	// mouth for it to eat.
	EatReachCols = 1.5
	EatReachRows = 2.0
	// MouthOpenReach is how close the Fish is to the Food it is going for when
	// it opens its mouth.
	MouthOpenReach = 4.0
	// InterceptSteps refines where the Fish aims for sinking Food: each step
	// predicts where the Food will be when the Fish gets there.
	InterceptSteps = 3
)

// energy is how lively the Fish is in one Mood.
type energy struct {
	// speed scales its top speed and acceleration when it wanders or darts.
	speed float64
	// dartChance is the chance that a trip is a quick dart.
	dartChance float64
	// pause scales how long it hovers between trips.
	pause float64
	// low keeps its wandering in the lower part of the Tank: the bottom
	// LowWater of the rows it can swim in.
	low bool
}

// energies is how lively the Fish is in each Mood: brisk when Content,
// slower when Hungry, and barely moving, low in the Tank, when Starving.
// Going for Food is as eager in any Mood.
var energies = map[fish.Mood]energy{
	fish.Content:  {speed: 1, dartChance: DartChance, pause: 1},
	fish.Hungry:   {speed: 0.5, dartChance: DartChance / 4, pause: 2},
	fish.Starving: {speed: 0.25, dartChance: 0, pause: 3, low: true},
}

// LowWater is the share of the rows the Fish can swim in, counted up from
// the bottom, that a Starving Fish keeps to.
const LowWater = 0.4

// DroopSpeed is the top speed at which a Starving Fish sinks to the lower
// part of the Tank. It is scaled by VerticalScale like any vertical speed.
const DroopSpeed = 8.0

// Blinking, bubbles, hearts and notes.
const (
	// BlinkMin and BlinkMax bound the time from one blink to the next, and
	// BlinkTime is how long the eyes stay shut.
	BlinkMin  = 3.0
	BlinkMax  = 6.0
	BlinkTime = 0.15

	// BubbleRise is how fast bubbles rise, in rows per second. A bubble is
	// small until it has risen BubbleMedium rows, and large after
	// BubbleLarge.
	BubbleRise   = 2.5
	BubbleMedium = 1.5
	BubbleLarge  = 4.0
	// AmbientMin and AmbientMax bound the time between bubbles rising from
	// the floor.
	AmbientMin = 0.6
	AmbientMax = 2.0
	// PuffMin and PuffMax bound the time between bubbles blown by a Content
	// Fish.
	PuffMin = 1.5
	PuffMax = 4.0

	// FlourishRise is how fast a heart or note floats up, in rows per
	// second. It lasts FlourishLife, fading for the last FlourishFade.
	FlourishRise = 1.2
	FlourishLife = 2.5
	FlourishFade = 1.0
	// NoteMin and NoteMax bound the time between the hearts and notes a
	// Content Fish shows now and then.
	NoteMin = 6.0
	NoteMax = 15.0
)
