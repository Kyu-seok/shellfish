package tank

// Tuning numbers for the Tank simulation. Distances are in cells, times in
// seconds. A cell is about twice as tall as it is wide, so vertical speeds are
// kept lower than horizontal ones.

// FishWidth and FishHeight are the size of the Fish's drawing, in cells.
const (
	FishWidth  = 10
	FishHeight = 3
)

const (
	// CruiseSpeed is the Fish's top speed on an ordinary trip.
	CruiseSpeed = 8.0
	// Accel is how quickly the Fish speeds up and slows down.
	Accel = 10.0
	// VerticalScale scales speeds and accelerations for up-and-down movement.
	VerticalScale = 0.4
	// TurnHold is how long the Fish holds still before it turns around.
	TurnHold = 0.35
	// ArriveDistance is how close to a wander target counts as arrived.
	ArriveDistance = 0.3
	// MaxTrip gives up on a trip that takes longer than this.
	MaxTrip = 15.0
	// TargetTries is how many random wander targets to try before giving up
	// and hovering a little longer.
	TargetTries = 30

	// SeaweedSpacing is about how many columns of Tank there are per stalk.
	SeaweedSpacing = 18
	// SeaweedMinHeight and SeaweedMaxHeight bound a stalk's height in rows.
	SeaweedMinHeight = 2
	SeaweedMaxHeight = 5
	// SeaweedSwayPeriod is how long a stalk takes to sway one way and back.
	SeaweedSwayPeriod = 1.6
	// FishClearance is the rows kept free above the tallest stalk so the Fish
	// can always swim over it.
	FishClearance = 2

	// TailBaseRate is how many times a second the tail swishes while still.
	TailBaseRate = 1.5
	// TailRatePerSpeed adds swishes per second for each cell per second of speed.
	TailRatePerSpeed = 0.8
)
