package airport

// Route represents an approach or departure route
type Route struct {
	Name      string     // Route name (e.g., "DERUP1A")
	Type      string     // "STAR", "SID", "APPROACH"
	Waypoints []string   // Ordered list of waypoint names
	Runway    string     // Associated runway (e.g., "13R")
}

// GetLHBPRoutes returns the STAR routes for LHBP
func GetLHBPRoutes() []Route {
	return []Route{
		// Northern STAR to RWY 13R
		{
			Name:      "DERUP1A",
			Type:      "STAR",
			Waypoints: []string{"DERUP", "GERSA", "OLBEN", "BP502", "BP513"},
			Runway:    "13R",
		},
		// Eastern STAR to RWY 13L
		{
			Name:      "ODINA1A",
			Type:      "STAR",
			Waypoints: []string{"ODINA", "NEMKI", "OSPEN", "BP503", "BP514"},
			Runway:    "13L",
		},
		// Southern STAR to RWY 13L
		{
			Name:      "RUSIK1A",
			Type:      "STAR",
			Waypoints: []string{"RUSIK", "VOSUD", "LUNIP", "BP503", "BP514"},
			Runway:    "13L",
		},
		// Western STAR to RWY 13R
		{
			Name:      "RIPMO1A",
			Type:      "STAR",
			Waypoints: []string{"RIPMO", "NEPOD", "PEPOK", "BP502", "BP513"},
			Runway:    "13R",
		},
	}
}

// GetLHBPSIDs returns the SID routes for LHBP
func GetLHBPSIDs() []Route {
	return []Route{
		{Name: "MASUN1A", Type: "SID", Waypoints: []string{"MASUN"}, Runway: "13R"},
		{Name: "RUDNO1A", Type: "SID", Waypoints: []string{"RUDNO"}, Runway: "13R"},
		{Name: "KEKEC1A", Type: "SID", Waypoints: []string{"KEKEC"}, Runway: "13L"},
		{Name: "BEREK1A", Type: "SID", Waypoints: []string{"BEREK"}, Runway: "13L"},
	}
}

// GetJFKSIDs returns the SID routes for KJFK
func GetJFKSIDs() []Route {
	return []Route{
		{Name: "GREKI1", Type: "SID", Waypoints: []string{"GREKI"}, Runway: "04L"},
		{Name: "YNKEE1", Type: "SID", Waypoints: []string{"YNKEE"}, Runway: "04R"},
		{Name: "DIXIE1", Type: "SID", Waypoints: []string{"DIXIE"}, Runway: "22R"},
		{Name: "ELIOT1", Type: "SID", Waypoints: []string{"ELIOT"}, Runway: "22L"},
	}
}

// GetLAXSIDs returns the SID routes for KLAX
func GetLAXSIDs() []Route {
	return []Route{
		{Name: "DOTSS1", Type: "SID", Waypoints: []string{"DOTSS"}, Runway: "24R"},
		{Name: "REYES1", Type: "SID", Waypoints: []string{"REYES"}, Runway: "24L"},
		{Name: "SNDGG1", Type: "SID", Waypoints: []string{"SNDGG"}, Runway: "06L"},
		{Name: "FILLM1", Type: "SID", Waypoints: []string{"FILLM"}, Runway: "06R"},
	}
}

// GetRoutes returns routes for a given airport
func GetRoutes(icao string) []Route {
	switch icao {
	case "LHBP":
		routes := GetLHBPRoutes()
		routes = append(routes, GetLHBPSIDs()...)
		return routes
	case "KJFK":
		return GetJFKSIDs()
	case "KLAX":
		return GetLAXSIDs()
	default:
		return []Route{}
	}
}

// GetSIDExits returns the SID exit waypoint names for a given airport
func GetSIDExits(icao string) []string {
	switch icao {
	case "LHBP":
		return []string{"MASUN", "RUDNO", "KEKEC", "BEREK"}
	case "KJFK":
		return []string{"GREKI", "YNKEE", "DIXIE", "ELIOT"}
	case "KLAX":
		return []string{"DOTSS", "REYES", "SNDGG", "FILLM"}
	default:
		return []string{}
	}
}
