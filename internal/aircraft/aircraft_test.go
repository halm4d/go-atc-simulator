package aircraft

import (
	"math"
	"testing"
)

func init() {
	// Register a minimal B738 type for tests so they don't depend on the data loader.
	if len(AircraftTypes) == 0 {
		RegisterTypes([]Type{
			{
				ICAO: "B738", Name: "Boeing 737-800",
				CruiseSpeed: 450, MaxSpeed: 490, MinSpeed: 140,
				MaxAltitude: 41000, ClimbRate: 2000, DescentRate: 2500,
				TurnRate: 3.0, WakeTurbulence: "M", Category: "JET",
			},
		})
	}
}

// ---- turn direction ----

func TestGetTurnDirection_Right(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.TargetHeading = 90
	dir := a.getTurnDirection()
	if dir != 1 {
		t.Errorf("expected right turn (+1), got %v", dir)
	}
}

func TestGetTurnDirection_Left(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 90, 250)
	a.TargetHeading = 0
	dir := a.getTurnDirection()
	if dir != -1 {
		t.Errorf("expected left turn (-1), got %v", dir)
	}
}

func TestGetTurnDirection_ShortestPath_CrossNorth(t *testing.T) {
	// From 350° to 010° — shortest is right (+20°), not left (-340°)
	a := NewAircraft("T1", "B738", 0, 0, 10000, 350, 250)
	a.TargetHeading = 10
	dir := a.getTurnDirection()
	if dir != 1 {
		t.Errorf("expected right turn (+1) crossing north, got %v", dir)
	}
}

func TestGetTurnDirection_ShortestPath_CrossSouth(t *testing.T) {
	// From 010° to 350° — shortest is left (-20°)
	a := NewAircraft("T1", "B738", 0, 0, 10000, 10, 250)
	a.TargetHeading = 350
	dir := a.getTurnDirection()
	if dir != -1 {
		t.Errorf("expected left turn (-1) crossing north, got %v", dir)
	}
}

// ---- altitude commands ----

func TestCommandAltitude_ClampToMax(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.CommandAltitude(99999)
	if a.TargetAltitude > a.Type.MaxAltitude {
		t.Errorf("altitude should be clamped to MaxAltitude (%v), got %v", a.Type.MaxAltitude, a.TargetAltitude)
	}
}

func TestCommandAltitude_ClampToZero(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.CommandAltitude(-500)
	if a.TargetAltitude < 0 {
		t.Errorf("altitude should be clamped to 0, got %v", a.TargetAltitude)
	}
}

// ---- speed commands ----

func TestCommandSpeed_ClampToMax(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.CommandSpeed(99999)
	if a.TargetSpeed > a.Type.MaxSpeed {
		t.Errorf("speed clamped to MaxSpeed (%v), got %v", a.Type.MaxSpeed, a.TargetSpeed)
	}
}

func TestCommandSpeed_ClampToMin(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.CommandSpeed(0)
	if a.TargetSpeed < a.Type.MinSpeed {
		t.Errorf("speed should be clamped to MinSpeed (%v), got %v", a.Type.MinSpeed, a.TargetSpeed)
	}
}

// ---- heading commands ----

func TestCommandHeading_NormalisesNegative(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.CommandHeading(-10)
	if a.TargetHeading < 0 || a.TargetHeading >= 360 {
		t.Errorf("heading should be in [0,360), got %v", a.TargetHeading)
	}
}

func TestCommandHeading_Normalises360(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.CommandHeading(360)
	if a.TargetHeading != 0 {
		t.Errorf("heading 360 should normalise to 0, got %v", a.TargetHeading)
	}
}

// ---- airborne physics ----

func TestUpdateAirborne_ClimbsTowardTarget(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 5000, 0, 250)
	a.Phase = PhaseArrival
	a.TargetAltitude = 10000
	a.Update(60) // 1 minute
	if a.Altitude <= 5000 {
		t.Errorf("aircraft should have climbed above 5000ft after 60s, got %.0f", a.Altitude)
	}
}

func TestUpdateAirborne_DescendsTowardTarget(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.Phase = PhaseArrival
	a.TargetAltitude = 5000
	a.Update(60)
	if a.Altitude >= 10000 {
		t.Errorf("aircraft should have descended below 10000ft after 60s, got %.0f", a.Altitude)
	}
}

func TestUpdateAirborne_TurnsTowardTarget(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.Phase = PhaseArrival
	a.TargetHeading = 90
	a.Update(10) // 10 seconds
	if a.Heading <= 0 {
		t.Errorf("aircraft should have turned right toward 090, heading still at %.0f", a.Heading)
	}
}

func TestUpdateAirborne_MovesForward(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 90, 300) // heading east
	a.Phase = PhaseArrival
	prevX := a.X
	a.Update(60) // 1 minute at 300 kts = 5nm
	if a.X <= prevX {
		t.Errorf("aircraft heading east should increase X, went from %.2f to %.2f", prevX, a.X)
	}
}

// ---- distance / separation helpers ----

func TestDistanceTo(t *testing.T) {
	a1 := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a2 := NewAircraft("T2", "B738", 3, 4, 10000, 0, 250)
	d := a1.DistanceTo(a2)
	if math.Abs(d-5.0) > 0.001 {
		t.Errorf("expected distance 5nm (3-4-5 triangle), got %.4f", d)
	}
}

func TestAltitudeSeparation(t *testing.T) {
	a1 := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a2 := NewAircraft("T2", "B738", 0, 0, 8500, 0, 250)
	sep := a1.AltitudeSeparation(a2)
	if math.Abs(sep-1500) > 0.01 {
		t.Errorf("expected 1500ft altitude separation, got %.2f", sep)
	}
}

// ---- holding pattern ----

func TestEnterHold_SetsPhase(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 180, 250)
	a.Phase = PhaseArrival
	a.EnterHold()
	if a.Phase != PhaseHolding {
		t.Errorf("expected PhaseHolding after EnterHold, got %v", a.Phase)
	}
}

func TestEnterHold_InboundIsReciprocal(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 90, 250)
	a.Phase = PhaseArrival
	a.EnterHold()
	// Aircraft heading 090 → inbound should be 270
	if a.HoldInboundHeading != 270 {
		t.Errorf("expected inbound heading 270, got %.0f", a.HoldInboundHeading)
	}
}

func TestEnterHold_StartsOutbound(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.Phase = PhaseArrival
	a.EnterHold()
	if a.HoldLeg != 1 {
		t.Errorf("expected HoldLeg=1 (outbound) at entry, got %d", a.HoldLeg)
	}
}

// ---- route following ----

func TestAdvanceRoute_SteersTowardWaypoint(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.Phase = PhaseArrival
	a.RouteWaypoints = [][2]float64{{10, 0}, {20, 0}}
	a.RouteNames = []string{"WP1", "WP2"}
	a.HasRoute = true
	a.RouteName = "STAR1"
	a.DirectTarget = "WP1"

	a.advanceRoute()

	// Aircraft at (0,0), waypoint at (10,0) → heading should be ~090
	if a.TargetHeading < 85 || a.TargetHeading > 95 {
		t.Errorf("expected heading ~090 toward waypoint, got %.1f", a.TargetHeading)
	}
}

func TestAdvanceRoute_PopsWaypointWhenReached(t *testing.T) {
	a := NewAircraft("T1", "B738", 9.5, 0, 10000, 90, 250)
	a.Phase = PhaseArrival
	a.RouteWaypoints = [][2]float64{{10, 0}, {20, 0}}
	a.RouteNames = []string{"WP1", "WP2"}
	a.HasRoute = true

	a.advanceRoute()

	if len(a.RouteWaypoints) != 1 {
		t.Fatalf("expected 1 waypoint remaining, got %d", len(a.RouteWaypoints))
	}
	if a.RouteNames[0] != "WP2" {
		t.Errorf("expected next waypoint WP2, got %s", a.RouteNames[0])
	}
	if a.DirectTarget != "WP2" {
		t.Errorf("expected DirectTarget WP2, got %s", a.DirectTarget)
	}
}

func TestAdvanceRoute_ClearsRouteWhenComplete(t *testing.T) {
	a := NewAircraft("T1", "B738", 9.5, 0, 10000, 90, 250)
	a.Phase = PhaseArrival
	a.RouteWaypoints = [][2]float64{{10, 0}}
	a.RouteNames = []string{"WP1"}
	a.HasRoute = true

	a.advanceRoute()

	if a.HasRoute {
		t.Error("expected HasRoute to be false after reaching last waypoint")
	}
	if a.DirectTarget != "" {
		t.Errorf("expected empty DirectTarget, got %s", a.DirectTarget)
	}
}

func TestAdvanceRoute_SkippedWhenCommanded(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	a.Phase = PhaseArrival
	a.RouteWaypoints = [][2]float64{{10, 0}}
	a.RouteNames = []string{"WP1"}
	a.HasRoute = true
	a.Commanded = true

	origHeading := a.TargetHeading
	a.Update(1.0) // should NOT call advanceRoute because Commanded=true

	if a.TargetHeading != origHeading {
		t.Errorf("expected heading unchanged when Commanded, got %.1f", a.TargetHeading)
	}
}

// ---- data tag ----

func TestRotateDataTag_Cycles(t *testing.T) {
	a := NewAircraft("T1", "B738", 0, 0, 10000, 0, 250)
	for i := 0; i < 4; i++ {
		a.RotateDataTag()
	}
	if a.DataTagPos != 0 {
		t.Errorf("DataTagPos should wrap back to 0 after 4 rotations, got %d", a.DataTagPos)
	}
}
