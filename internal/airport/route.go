package airport

// Route represents an approach or departure route
type Route struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Waypoints []string `json:"waypoints"`
	Runway    string   `json:"runway"`
}

// routes is the registry of routes per airport.
var routes = map[string][]Route{}

// RegisterRoutes adds routes for an airport.
func RegisterRoutes(icao string, r []Route) {
	routes[icao] = r
}

// GetRoutes returns routes for a given airport
func GetRoutes(icao string) []Route {
	return routes[icao]
}

// GetSIDExits returns the SID exit waypoint names for a given airport,
// derived from the last waypoint of each SID route.
func GetSIDExits(icao string) []string {
	var exits []string
	for _, r := range routes[icao] {
		if r.Type == "SID" && len(r.Waypoints) > 0 {
			exits = append(exits, r.Waypoints[len(r.Waypoints)-1])
		}
	}
	return exits
}
