package airport

import "math"

// Airport represents an airport with its runways and position
type Airport struct {
	ICAO      string   // ICAO code (e.g., "KJFK")
	IATA      string   // IATA code (e.g., "JFK")
	Name      string   // Full name
	Latitude  float64  // Latitude
	Longitude float64  // Longitude
	Elevation float64  // Elevation in feet
	Runways   []Runway // List of runways
}

// Runway represents a runway with its headings and position
type Runway struct {
	Name1    string  // First designation (e.g., "04L")
	Name2    string  // Opposite designation (e.g., "22R")
	Heading1 float64 // Heading for Name1 (degrees)
	Heading2 float64 // Heading for Name2 (degrees)
	Length   float64 // Length in feet
	Width    float64 // Width in feet
	// Simplified: runway starts at airport center, extends along heading
}

// John F. Kennedy International Airport (KJFK)
var KJFK = Airport{
	ICAO:      "KJFK",
	IATA:      "JFK",
	Name:      "John F. Kennedy International Airport",
	Latitude:  40.6413,
	Longitude: -73.7781,
	Elevation: 13,
	Runways: []Runway{
		{
			Name1:    "04L",
			Name2:    "22R",
			Heading1: 40,
			Heading2: 220,
			Length:   11351,
			Width:    150,
		},
		{
			Name1:    "04R",
			Name2:    "22L",
			Heading1: 40,
			Heading2: 220,
			Length:   8400,
			Width:    200,
		},
		{
			Name1:    "13L",
			Name2:    "31R",
			Heading1: 130,
			Heading2: 310,
			Length:   10000,
			Width:    150,
		},
		{
			Name1:    "13R",
			Name2:    "31L",
			Heading1: 130,
			Heading2: 310,
			Length:   14511,
			Width:    200,
		},
	},
}

// Los Angeles International Airport (KLAX)
var KLAX = Airport{
	ICAO:      "KLAX",
	IATA:      "LAX",
	Name:      "Los Angeles International Airport",
	Latitude:  33.9416,
	Longitude: -118.4085,
	Elevation: 125,
	Runways: []Runway{
		{
			Name1:    "06L",
			Name2:    "24R",
			Heading1: 69,
			Heading2: 249,
			Length:   8926,
			Width:    150,
		},
		{
			Name1:    "06R",
			Name2:    "24L",
			Heading1: 69,
			Heading2: 249,
			Length:   10285,
			Width:    150,
		},
		{
			Name1:    "07L",
			Name2:    "25R",
			Heading1: 73,
			Heading2: 253,
			Length:   11095,
			Width:    150,
		},
		{
			Name1:    "07R",
			Name2:    "25L",
			Heading1: 73,
			Heading2: 253,
			Length:   12091,
			Width:    200,
		},
	},
}

// Budapest Ferenc Liszt International Airport (LHBP)
var LHBP = Airport{
	ICAO:      "LHBP",
	IATA:      "BUD",
	Name:      "Budapest Ferenc Liszt International Airport",
	Latitude:  47.4398,
	Longitude: 19.2618,
	Elevation: 495,
	Runways: []Runway{
		{
			Name1:    "13R",
			Name2:    "31L",
			Heading1: 131,
			Heading2: 311,
			Length:   10827,
			Width:    148,
		},
		{
			Name1:    "13L",
			Name2:    "31R",
			Heading1: 131,
			Heading2: 311,
			Length:   9843,
			Width:    148,
		},
	},
}

// GetAirport returns an airport by ICAO code
func GetAirport(icao string) *Airport {
	switch icao {
	case "KJFK":
		return &KJFK
	case "KLAX":
		return &KLAX
	case "LHBP":
		return &LHBP
	default:
		return &KJFK // Default to JFK
	}
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
	x = halfLength * math.Cos(headingRad)
	y = halfLength * math.Sin(headingRad)
	return
}
