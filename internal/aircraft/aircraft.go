package aircraft

import (
	"atc-sim/internal/airport"
	"fmt"
	"math"
)

// FlightPhase represents the current phase of flight
type FlightPhase int

const (
	PhaseHoldingShort FlightPhase = iota // Holding short of runway, awaiting clearance
	PhaseLineUpWait                      // On runway, line up and wait
	PhaseTakeoffRoll                     // Accelerating on runway for takeoff
	PhaseClimbout                        // Airborne below 1500ft AGL (Tower frequency)
	PhaseDeparture                       // En-route departure (Departure Control)
	PhaseArrival                         // Arriving aircraft (being vectored by Approach)
	PhaseHolding                         // Flying a racetrack holding pattern
	PhaseFinal                           // Established on ILS, auto-descending (awaiting clearance)
	PhaseLanding                         // Cleared to land, continuing to touchdown
	PhaseLanded                          // Touched down — to be removed from simulation
)

// Aircraft represents an individual aircraft in the simulation
type Aircraft struct {
	Callsign string  // Flight callsign (e.g., "UAL123")
	Type     Type    // Aircraft type
	X        float64 // Longitude-based X position (nm from reference)
	Y        float64 // Latitude-based Y position (nm from reference)
	Altitude float64 // Altitude in feet
	Heading  float64 // Heading in degrees (0-360)
	Speed    float64 // Ground speed in knots

	// ATC commands
	TargetHeading  float64 // Commanded heading
	TargetAltitude float64 // Commanded altitude
	TargetSpeed    float64 // Commanded speed

	// State
	Phase            FlightPhase
	Selected         bool
	IsArrival        bool
	IsDeparture      bool
	Commanded        bool    // True once any ATC command has been issued
	DirectTarget     string  // Named direct-to target (waypoint/runway), "" if none
	RunwayName       string  // Assigned runway (e.g., "13R")
	RunwayHeading    float64 // Takeoff / approach heading in degrees
	AirportElevation float64 // Airport elevation in feet
	ThresholdX       float64 // Runway threshold X position (nm from airport center)
	ThresholdY       float64 // Runway threshold Y position (nm from airport center)
	DataTagOffX      int     // Tag X offset from aircraft screen position
	DataTagOffY      int     // Tag Y offset from aircraft screen position
	Trail      [][2]float64 // Recent position samples for trail rendering
	TrailTimer float64      // Accumulator for trail sampling interval

	// Holding pattern state
	HoldInboundHeading float64 // Inbound heading toward fix
	HoldLeg            int     // 0=inbound, 1=outbound
	HoldLegTimer       float64 // Seconds spent in current leg
	HoldLegDuration    float64 // Target seconds per leg (based on speed)

	// Route following (STAR/SID — pre-resolved coordinates set by game.go)
	RouteWaypoints [][2]float64 // Remaining waypoint coordinates to follow
	RouteNames     []string     // Parallel slice of waypoint names (for display)
	HasRoute       bool         // True when actively following a route
	RouteName      string       // Route name for display (e.g., "DERUP1A")
	AssignedRoute  string       // Originally assigned STAR/SID name (never cleared)

	// Pilot request flags (for chat system)
	HasRequestedDeparture    bool
	HasRequestedLanding      bool
	HasRequestedInstructions bool
	PrevHasRoute             bool    // For detecting route-completion transitions
	HoldShortTimer           float64 // Seconds spent in PhaseHoldingShort
}

// NewAircraft creates a new aircraft
func NewAircraft(callsign string, typeCode string, x, y, altitude, heading, speed float64) *Aircraft {
	aircraftType, exists := GetType(typeCode)
	if !exists {
		// Default to B738 if type not found
		aircraftType = AircraftTypes["B738"]
	}

	return &Aircraft{
		Callsign:       callsign,
		Type:           aircraftType,
		X:              x,
		Y:              y,
		Altitude:       altitude,
		Heading:        heading,
		Speed:          speed,
		TargetHeading:  heading,
		TargetAltitude: altitude,
		TargetSpeed:    speed,
		Phase:          PhaseArrival, // default; departure spawning overrides this
		Selected:       false,
		DataTagOffX:    DefaultDataTagOffX,
		DataTagOffY:    DefaultDataTagOffY,
	}
}

// IsCommandable returns true when the aircraft can receive H/A/S/D commands
func (a *Aircraft) IsCommandable() bool {
	return a.Phase == PhaseArrival || a.Phase == PhaseDeparture ||
		a.Phase == PhaseClimbout || a.Phase == PhaseFinal ||
		a.Phase == PhaseHolding
}

// EnterHold places the aircraft into a standard right-hand racetrack holding pattern.
// The inbound heading is the reciprocal of the aircraft's current heading (fly away first).
func (a *Aircraft) EnterHold() {
	// Outbound heading = current heading; inbound = reciprocal
	a.HoldInboundHeading = airport.NormalizeHeading(a.Heading + 180)
	// Leg duration: standard 1-minute legs (60s) adjusted for speed
	// At 250 kts the default 1-min leg is fine; scale proportionally
	a.HoldLegDuration = HoldLegDuration
	a.HoldLeg = 1 // start on outbound leg (flying current heading away from fix)
	a.HoldLegTimer = 0
	a.Phase = PhaseHolding
	a.DirectTarget = ""
}

// Update updates the aircraft position and state based on physics
func (a *Aircraft) Update(deltaTime float64) {
	switch a.Phase {
	case PhaseHoldingShort, PhaseLineUpWait, PhaseLanded:
		// Stationary — no movement
		return
	case PhaseTakeoffRoll:
		a.updateTakeoffRoll(deltaTime)
	case PhaseHolding:
		a.updateHolding(deltaTime)
	case PhaseFinal, PhaseLanding:
		a.updateFinalApproach(deltaTime)
	default:
		a.updateAirborne(deltaTime)
	}

	// Sample position for trail every TrailSampleInterval seconds of sim time
	a.TrailTimer += deltaTime
	if a.TrailTimer >= TrailSampleInterval {
		a.TrailTimer = 0
		a.Trail = append(a.Trail, [2]float64{a.X, a.Y})
		if len(a.Trail) > MaxTrailPoints {
			a.Trail = a.Trail[1:]
		}
	}
}

// updateHolding flies a standard right-hand racetrack holding pattern.
// Leg 0 = inbound, Leg 1 = outbound. The aircraft turns right at each end.
func (a *Aircraft) updateHolding(deltaTime float64) {
	a.HoldLegTimer += deltaTime

	outboundHeading := airport.NormalizeHeading(a.HoldInboundHeading + 180)

	switch a.HoldLeg {
	case 0: // inbound leg
		a.TargetHeading = a.HoldInboundHeading
		if a.HoldLegTimer >= a.HoldLegDuration {
			// End of inbound: switch to outbound (right turn)
			a.HoldLeg = 1
			a.HoldLegTimer = 0
		}
	case 1: // outbound leg
		a.TargetHeading = outboundHeading
		if a.HoldLegTimer >= a.HoldLegDuration {
			// End of outbound: switch back to inbound (right turn)
			a.HoldLeg = 0
			a.HoldLegTimer = 0
		}
	}

	a.updateAirborne(deltaTime)
}

// updateTakeoffRoll handles physics during the takeoff roll
func (a *Aircraft) updateTakeoffRoll(deltaTime float64) {
	rotateSpeed := a.Type.MinSpeed * RotateSpeedFactor

	// Accelerate along runway (~4 knots/sec, roughly realistic for a jet)
	a.Speed += TakeoffAccelRate * deltaTime

	// Lock heading to runway heading during roll
	a.Heading = a.RunwayHeading
	a.TargetHeading = a.RunwayHeading

	// Move along runway
	headingRad := (90 - a.RunwayHeading) * math.Pi / 180
	distance := a.Speed * deltaTime / 3600.0
	a.X += distance * math.Cos(headingRad)
	a.Y += distance * math.Sin(headingRad)

	// At rotate speed, start climbing
	if a.Speed >= rotateSpeed {
		a.Altitude += a.Type.ClimbRate * deltaTime / 60.0
	}

	// Transition to climbout once airborne
	if a.Altitude > a.AirportElevation+AirborneAltThreshold {
		a.Phase = PhaseClimbout
		a.TargetSpeed = a.Type.MinSpeed * GoAroundSpeedFactor
		a.TargetAltitude = a.AirportElevation + InitialClimbAlt
	}
}

// updateFinalApproach handles ILS glide-slope descent and landing
func (a *Aircraft) updateFinalApproach(deltaTime float64) {
	// Lock onto the runway heading
	a.TargetHeading = a.RunwayHeading

	// Slow down toward approach speed
	approachSpeed := a.Type.MinSpeed * ApproachSpeedFactor
	if a.TargetSpeed > approachSpeed {
		a.TargetSpeed = approachSpeed
	}

	// Run normal airborne physics (turn, climb/descend, move)
	a.updateAirborne(deltaTime)

	// Distance to threshold (recalculated after position update)
	dx := a.X - a.ThresholdX
	dy := a.Y - a.ThresholdY
	distToThreshold := math.Sqrt(dx*dx + dy*dy)

	// Follow the 3° ILS glide slope (318 ft per nm)
	glideslopeAlt := a.AirportElevation + distToThreshold*GlideSlopeGradient
	if glideslopeAlt < a.Altitude {
		a.TargetAltitude = glideslopeAlt
	}

	// Not cleared — go around if within 2 nm of threshold
	if a.Phase == PhaseFinal && distToThreshold < GoAroundDistanceNm {
		a.Phase = PhaseArrival
		a.TargetAltitude = a.AirportElevation + GoAroundAltFt
		a.TargetSpeed = a.Type.MinSpeed * GoAroundSpeedFactor
		return
	}

	// Touchdown when cleared and within 0.2 nm of threshold (touchdown zone)
	if a.Phase == PhaseLanding && distToThreshold < TouchdownDistanceNm {
		a.Phase = PhaseLanded
	}
}

// updateAirborne handles normal airborne flight physics
func (a *Aircraft) updateAirborne(deltaTime float64) {
	// Update heading (turn toward target)
	if math.Abs(a.Heading-a.TargetHeading) > HeadingTolerance {
		turnDirection := a.getTurnDirection()
		turnAmount := a.Type.TurnRate * deltaTime
		a.Heading += turnDirection * turnAmount

		a.Heading = airport.NormalizeHeading(a.Heading)
	}

	// Update altitude (climb/descend toward target)
	altDiff := a.TargetAltitude - a.Altitude
	if math.Abs(altDiff) > AltitudeTolerance {
		var verticalSpeed float64
		if altDiff > 0 {
			verticalSpeed = a.Type.ClimbRate
		} else {
			verticalSpeed = -a.Type.DescentRate
		}
		a.Altitude += verticalSpeed * deltaTime / 60.0
	}

	// Update speed (accelerate/decelerate toward target)
	speedDiff := a.TargetSpeed - a.Speed
	if math.Abs(speedDiff) > SpeedTolerance {
		accelRate := AccelRate
		if speedDiff > 0 {
			a.Speed += accelRate * deltaTime
			if a.Speed > a.TargetSpeed {
				a.Speed = a.TargetSpeed
			}
		} else {
			a.Speed -= accelRate * deltaTime
			if a.Speed < a.TargetSpeed {
				a.Speed = a.TargetSpeed
			}
		}

		// Enforce speed limits
		if a.Speed > a.Type.MaxSpeed {
			a.Speed = a.Type.MaxSpeed
		}
		if a.Speed < a.Type.MinSpeed {
			a.Speed = a.Type.MinSpeed
		}
	}

	// Update position based on heading and speed
	headingRad := (90 - a.Heading) * math.Pi / 180
	distance := a.Speed * deltaTime / 3600.0
	a.X += distance * math.Cos(headingRad)
	a.Y += distance * math.Sin(headingRad)

	// Transition from climbout to departure once above 1500ft AGL
	if a.Phase == PhaseClimbout && a.Altitude > a.AirportElevation+ClimboutAGLThreshold {
		a.Phase = PhaseDeparture
	}

	// Route auto-steering: follow STAR/SID waypoints when not manually commanded.
	if a.HasRoute && !a.Commanded {
		a.advanceRoute()
	}
}

// advanceRoute steers toward the next waypoint in RouteWaypoints.
// When a waypoint is reached (within 2 nm), it is popped and the next one becomes active.
func (a *Aircraft) advanceRoute() {
	if len(a.RouteWaypoints) == 0 {
		a.HasRoute = false
		a.DirectTarget = ""
		return
	}

	targetX := a.RouteWaypoints[0][0]
	targetY := a.RouteWaypoints[0][1]
	dx := targetX - a.X
	dy := targetY - a.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist < WaypointArrivalNm {
		// Reached this waypoint — pop it
		a.RouteWaypoints = a.RouteWaypoints[1:]
		a.RouteNames = a.RouteNames[1:]
		if len(a.RouteWaypoints) == 0 {
			a.HasRoute = false
			a.DirectTarget = ""
			return
		}
		a.DirectTarget = a.RouteNames[0]
		return
	}

	// Steer toward current waypoint
	hdg := airport.NormalizeHeading(90 - math.Atan2(dy, dx)*180/math.Pi)
	a.TargetHeading = hdg
}

// getTurnDirection returns -1 for left turn, 1 for right turn (shortest path)
func (a *Aircraft) getTurnDirection() float64 {
	diff := a.TargetHeading - a.Heading

	// Normalize to -180 to 180
	for diff > 180 {
		diff -= 360
	}
	for diff < -180 {
		diff += 360
	}

	if diff > 0 {
		return 1
	}
	return -1
}

// CommandHeading sets a new heading command
func (a *Aircraft) CommandHeading(heading float64) {
	a.TargetHeading = airport.NormalizeHeading(heading)
}

// CommandAltitude sets a new altitude command
func (a *Aircraft) CommandAltitude(altitude float64) {
	if altitude > a.Type.MaxAltitude {
		altitude = a.Type.MaxAltitude
	}
	if altitude < 0 {
		altitude = 0
	}
	a.TargetAltitude = altitude
}

// CommandSpeed sets a new speed command
func (a *Aircraft) CommandSpeed(speed float64) {
	if speed > a.Type.MaxSpeed {
		speed = a.Type.MaxSpeed
	}
	if speed < a.Type.MinSpeed {
		speed = a.Type.MinSpeed
	}
	a.TargetSpeed = speed
}

// GetDataTag returns the formatted data tag for display
func (a *Aircraft) GetDataTag() string {
	altHundreds := int(a.Altitude / 100)
	speed := int(a.Speed)
	return fmt.Sprintf("%s\n%s %03d %d↑", a.Callsign, a.Type.ICAO, altHundreds, speed)
}

// DistanceTo calculates the distance to another aircraft in nautical miles
func (a *Aircraft) DistanceTo(other *Aircraft) float64 {
	dx := a.X - other.X
	dy := a.Y - other.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// AltitudeSeparation returns the altitude separation in feet
func (a *Aircraft) AltitudeSeparation(other *Aircraft) float64 {
	return math.Abs(a.Altitude - other.Altitude)
}

// ResetDataTag resets the data tag offset to the default top-right position.
func (a *Aircraft) ResetDataTag() {
	a.DataTagOffX = DefaultDataTagOffX
	a.DataTagOffY = DefaultDataTagOffY
}
