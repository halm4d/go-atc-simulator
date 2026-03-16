package aircraft

const (
	// Trail sampling
	TrailSampleInterval = 5.0 // seconds between trail samples
	MaxTrailPoints      = 15

	// Physics
	AccelRate           = 10.0  // knots per second
	TakeoffAccelRate    = 4.0   // knots per second during takeoff roll
	AirborneAltThreshold = 50.0 // feet above airport elevation to be "airborne"
	InitialClimbAlt     = 5000.0 // feet above airport for initial climb target
	ClimboutAGLThreshold = 1500.0 // feet AGL to transition from climbout to departure

	// Approach
	GlideSlopeGradient   = 318.0 // ft per nm (3° ILS)
	GoAroundDistanceNm   = 2.0
	GoAroundAltFt        = 3000.0
	GoAroundSpeedFactor  = 1.3
	TouchdownDistanceNm  = 0.2
	ApproachSpeedFactor  = 1.05
	RotateSpeedFactor    = 1.1

	// Route following
	WaypointArrivalNm = 2.0 // nm threshold to consider waypoint reached

	// Data tag defaults
	DefaultDataTagOffX = 15
	DefaultDataTagOffY = -42

	// Heading
	HeadingTolerance = 0.5 // degrees — below this, heading is "aligned"

	// Altitude
	AltitudeTolerance = 50.0 // feet — below this, altitude is "at target"

	// Speed
	SpeedTolerance = 1.0 // knots — below this, speed is "at target"

	// Holding pattern
	HoldLegDuration = 60.0 // seconds per holding leg
)
