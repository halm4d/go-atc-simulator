package aircraft

// Type represents an aircraft type with its performance characteristics
type Type struct {
	ICAO           string  `json:"icao"`
	Name           string  `json:"name"`
	CruiseSpeed    float64 `json:"cruiseSpeed"`
	MaxSpeed       float64 `json:"maxSpeed"`
	MinSpeed       float64 `json:"minSpeed"`
	MaxAltitude    float64 `json:"maxAltitude"`
	ClimbRate      float64 `json:"climbRate"`
	DescentRate    float64 `json:"descentRate"`
	TurnRate       float64 `json:"turnRate"`
	WakeTurbulence string  `json:"wakeTurbulence"`
	Category       string  `json:"category"`
}

// AircraftTypes is the registry of known aircraft types, populated by the data loader.
var AircraftTypes = map[string]Type{}

// RegisterTypes populates the aircraft type registry from a slice of types.
func RegisterTypes(types []Type) {
	for _, t := range types {
		AircraftTypes[t.ICAO] = t
	}
}

// GetType returns the aircraft type by ICAO code
func GetType(icao string) (Type, bool) {
	t, exists := AircraftTypes[icao]
	return t, exists
}
