package airport

import (
	"math"
	"strings"
)

// Waypoint represents a navigation fix
type Waypoint struct {
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Type string  `json:"type"`
}

// waypoints is the registry of waypoints per airport.
var waypoints = map[string][]Waypoint{}

// RegisterWaypoints adds waypoints for an airport.
func RegisterWaypoints(icao string, w []Waypoint) {
	waypoints[icao] = w
}

// GetWaypoints returns waypoints for a given airport
func GetWaypoints(icao string) []Waypoint {
	return waypoints[icao]
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
