package atc

import (
	"atc-sim/internal/aircraft"
)

const (
	// Standard separation minimums
	MinHorizontalSeparation = 5.0  // Nautical miles
	MinVerticalSeparation   = 1000 // Feet
)

// Conflict represents a separation conflict between two aircraft
type Conflict struct {
	Aircraft1          *aircraft.Aircraft
	Aircraft2          *aircraft.Aircraft
	HorizontalDistance float64
	VerticalDistance   float64
	Severity           string // "WARNING" or "CRITICAL"
}

// CheckSeparation checks for separation violations between all aircraft
func CheckSeparation(aircraftList []*aircraft.Aircraft) []Conflict {
	var conflicts []Conflict

	for i := 0; i < len(aircraftList); i++ {
		for j := i + 1; j < len(aircraftList); j++ {
			a1 := aircraftList[i]
			a2 := aircraftList[j]

			// Ground aircraft are not subject to airborne separation checks
			if a1.Phase == aircraft.PhaseHoldingShort || a1.Phase == aircraft.PhaseLineUpWait ||
				a2.Phase == aircraft.PhaseHoldingShort || a2.Phase == aircraft.PhaseLineUpWait {
				continue
			}

			horizDist := a1.DistanceTo(a2)
			vertDist := a1.AltitudeSeparation(a2)

			// Check if separation is violated
			if horizDist < MinHorizontalSeparation && vertDist < MinVerticalSeparation {
				severity := "WARNING"
				if horizDist < 3.0 && vertDist < 500 {
					severity = "CRITICAL"
				}

				conflicts = append(conflicts, Conflict{
					Aircraft1:          a1,
					Aircraft2:          a2,
					HorizontalDistance: horizDist,
					VerticalDistance:   vertDist,
					Severity:           severity,
				})
			}
		}
	}

	return conflicts
}

// PredictConflict predicts if two aircraft will have a conflict in the near future
func PredictConflict(a1, a2 *aircraft.Aircraft, lookAheadSeconds float64) bool {
	// Simple prediction: extrapolate positions based on current heading and speed
	// This is a simplified version - real ATC uses more sophisticated algorithms

	steps := 10
	dt := lookAheadSeconds / float64(steps)

	// Create temporary copies
	temp1 := *a1
	temp2 := *a2

	for i := 0; i < steps; i++ {
		temp1.Update(dt)
		temp2.Update(dt)

		horizDist := temp1.DistanceTo(&temp2)
		vertDist := temp1.AltitudeSeparation(&temp2)

		if horizDist < MinHorizontalSeparation && vertDist < MinVerticalSeparation {
			return true
		}
	}

	return false
}
