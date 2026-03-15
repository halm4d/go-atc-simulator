package airport

import (
	"math"
	"strings"
)

// Waypoint represents a navigation fix
type Waypoint struct {
	Name string  // Fix name (e.g., "TORAZ")
	X    float64 // Position X in nm from airport
	Y    float64 // Position Y in nm from airport
	Type string  // "IAF", "FAF", "VOR", etc.
}

// GetJFKWaypoints returns the IAF waypoints for JFK
func GetJFKWaypoints() []Waypoint {
	return []Waypoint{
		{Name: "TORAZ", X: -25, Y: 15, Type: "IAF"},
		{Name: "CATUZ", X: 15, Y: 25, Type: "IAF"},
		{Name: "ATICO", X: 25, Y: -10, Type: "IAF"},
		{Name: "NICRA", X: -20, Y: -20, Type: "IAF"},
		{Name: "HAPIE", X: -15, Y: 20, Type: "IAF"},
		{Name: "CRANK", X: 20, Y: 10, Type: "IAF"},
		// SID exit fixes (departure exits ~35nm out)
		{Name: "GREKI", X: 20, Y: 35, Type: "SID"},
		{Name: "YNKEE", X: 35, Y: -10, Type: "SID"},
		{Name: "DIXIE", X: -10, Y: -35, Type: "SID"},
		{Name: "ELIOT", X: -35, Y: 10, Type: "SID"},
	}
}

// GetLAXWaypoints returns the IAF waypoints for LAX
func GetLAXWaypoints() []Waypoint {
	return []Waypoint{
		{Name: "SADDE", X: -20, Y: 20, Type: "IAF"},
		{Name: "LIMAA", X: 20, Y: 15, Type: "IAF"},
		{Name: "DARTS", X: -15, Y: -20, Type: "IAF"},
		{Name: "BASET", X: 15, Y: -15, Type: "IAF"},
		// SID exit fixes
		{Name: "DOTSS", X: -35, Y: 0, Type: "SID"},
		{Name: "REYES", X: 0, Y: 35, Type: "SID"},
		{Name: "SNDGG", X: 30, Y: -25, Type: "SID"},
		{Name: "FILLM", X: -15, Y: -35, Type: "SID"},
	}
}

// GetLHBPWaypoints returns the STAR waypoints for LHBP (Budapest)
// Based on real STAR procedures for RWY 13R/13L
func GetLHBPWaypoints() []Waypoint {
	return []Waypoint{
		// Northern STAR (from Vienna/Bratislava direction)
		{
			Name: "DERUP",
			X:    -30,
			Y:    25,
			Type: "STAR",
		},
		{
			Name: "GERSA",
			X:    -20,
			Y:    18,
			Type: "STAR",
		},
		{
			Name: "OLBEN",
			X:    -12,
			Y:    12,
			Type: "IAF",
		},
		// Eastern STAR (from Romania direction)
		{
			Name: "ODINA",
			X:    30,
			Y:    15,
			Type: "STAR",
		},
		{
			Name: "NEMKI",
			X:    22,
			Y:    10,
			Type: "STAR",
		},
		{
			Name: "OSPEN",
			X:    15,
			Y:    8,
			Type: "IAF",
		},
		// Southern STAR (from Belgrade direction)
		{
			Name: "RUSIK",
			X:    15,
			Y:    -28,
			Type: "STAR",
		},
		{
			Name: "VOSUD",
			X:    10,
			Y:    -18,
			Type: "STAR",
		},
		{
			Name: "LUNIP",
			X:    8,
			Y:    -12,
			Type: "IAF",
		},
		// Western STAR (from Zagreb direction)
		{
			Name: "RIPMO",
			X:    -28,
			Y:    -12,
			Type: "STAR",
		},
		{
			Name: "NEPOD",
			X:    -20,
			Y:    -8,
			Type: "STAR",
		},
		{
			Name: "PEPOK",
			X:    -12,
			Y:    -6,
			Type: "IAF",
		},
		// Intermediate fixes
		{Name: "BP502", X: -8, Y: 5, Type: "IF"},
		{Name: "BP503", X: 5, Y: 3, Type: "IF"},
		// Final approach fixes
		{Name: "BP513", X: -4, Y: 2, Type: "FAF"},
		{Name: "BP514", X: 3, Y: 1, Type: "FAF"},
		// SID exit fixes (NW, NE, SW, SE)
		{Name: "MASUN", X: -25, Y: 28, Type: "SID"}, // NW — toward Vienna/Bratislava
		{Name: "RUDNO", X: 28, Y: 22, Type: "SID"},  // NE — toward Slovakia/Ukraine
		{Name: "KEKEC", X: -20, Y: -28, Type: "SID"}, // SW — toward Croatia/Zagreb
		{Name: "BEREK", X: 25, Y: -28, Type: "SID"},  // SE — toward Serbia/Romania
	}
}

// GetWaypoints returns waypoints for a given airport
func GetWaypoints(icao string) []Waypoint {
	switch icao {
	case "KJFK":
		return GetJFKWaypoints()
	case "KLAX":
		return GetLAXWaypoints()
	case "LHBP":
		return GetLHBPWaypoints()
	default:
		return GetJFKWaypoints()
	}
}

// FindWaypoint finds a waypoint by name (case-insensitive)
func FindWaypoint(waypoints []Waypoint, name string) *Waypoint {
	upper := strings.ToUpper(name)
	for i := range waypoints {
		if strings.ToUpper(waypoints[i].Name) == upper {
			return &waypoints[i]
		}
	}
	return nil
}

// DistanceBetween calculates distance between two points
func DistanceBetween(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return math.Sqrt(dx*dx + dy*dy)
}

// HeadingTo calculates heading from one point to another
func HeadingTo(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	heading := 90 - math.Atan2(dy, dx)*180/math.Pi
	if heading < 0 {
		heading += 360
	}
	if heading >= 360 {
		heading -= 360
	}
	return heading
}
