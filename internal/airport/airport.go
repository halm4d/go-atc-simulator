package airport

import "math"

// Airport represents an airport with its runways and position
type Airport struct {
	ICAO      string   `json:"icao"`
	IATA      string   `json:"iata"`
	Name      string   `json:"name"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Elevation float64  `json:"elevation"`
	Runways   []Runway `json:"runways"`
}

// Runway represents a runway with its headings and position
type Runway struct {
	Name1    string  `json:"name1"`
	Name2    string  `json:"name2"`
	Heading1 float64 `json:"heading1"`
	Heading2 float64 `json:"heading2"`
	Length   float64 `json:"length"`
	Width    float64 `json:"width"`
	CenterX  float64 `json:"centerX"` // nm east of airport reference point
	CenterY  float64 `json:"centerY"` // nm north of airport reference point
}

// airports is the registry of loaded airports.
var airports = map[string]*Airport{}

// RegisterAirport adds an airport to the registry.
func RegisterAirport(a *Airport) {
	airports[a.ICAO] = a
}

// GetAirport returns an airport by ICAO code
func GetAirport(icao string) *Airport {
	if a, ok := airports[icao]; ok {
		return a
	}
	// Default fallback to first registered airport, or nil
	if a, ok := airports["KJFK"]; ok {
		return a
	}
	for _, a := range airports {
		return a
	}
	return nil
}

// FindRunwayByName finds a runway by its designation (e.g., "13R" or "31L")
func (a *Airport) FindRunwayByName(name string) (*Runway, bool) {
	for i := range a.Runways {
		if a.Runways[i].Name1 == name || a.Runways[i].Name2 == name {
			return &a.Runways[i], true
		}
	}
	return nil, false
}

// GetRunwayThreshold returns the X,Y position of a runway threshold (nm from airport
// center) and the takeoff heading for the given runway designation.
func GetRunwayThreshold(runway *Runway, runwayName string) (x, y, takeoffHeading float64) {
	lengthNm := runway.Length / 6076.12
	halfLength := lengthNm / 2

	// The threshold of Name1 is at the far end in the Name2 direction, and vice-versa.
	var thresholdDirection float64
	if runway.Name1 == runwayName {
		thresholdDirection = runway.Heading2
		takeoffHeading = runway.Heading1
	} else {
		thresholdDirection = runway.Heading1
		takeoffHeading = runway.Heading2
	}

	headingRad := (90 - thresholdDirection) * math.Pi / 180
	x = runway.CenterX + halfLength*math.Cos(headingRad)
	y = runway.CenterY + halfLength*math.Sin(headingRad)
	return
}
