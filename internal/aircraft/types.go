package aircraft

// Type represents an aircraft type with its performance characteristics
type Type struct {
	ICAO             string  // ICAO aircraft type code (e.g., "B738")
	Name             string  // Full name (e.g., "Boeing 737-800")
	CruiseSpeed      float64 // Knots
	MaxSpeed         float64 // Knots
	MinSpeed         float64 // Knots
	MaxAltitude      float64 // Feet
	ClimbRate        float64 // Feet per minute
	DescentRate      float64 // Feet per minute
	TurnRate         float64 // Degrees per second
	WakeTurbulence   string  // "L" (Light), "M" (Medium), "H" (Heavy), "J" (Super)
	Category         string  // "JET", "TURBOPROP", "PROP"
}

// Common aircraft types used in commercial aviation
var AircraftTypes = map[string]Type{
	"B738": {
		ICAO:           "B738",
		Name:           "Boeing 737-800",
		CruiseSpeed:    450,
		MaxSpeed:       490,
		MinSpeed:       140,
		MaxAltitude:    41000,
		ClimbRate:      2000,
		DescentRate:    2500,
		TurnRate:       3.0,
		WakeTurbulence: "M",
		Category:       "JET",
	},
	"A320": {
		ICAO:           "A320",
		Name:           "Airbus A320",
		CruiseSpeed:    450,
		MaxSpeed:       490,
		MinSpeed:       138,
		MaxAltitude:    39800,
		ClimbRate:      2000,
		DescentRate:    2500,
		TurnRate:       3.0,
		WakeTurbulence: "M",
		Category:       "JET",
	},
	"B77W": {
		ICAO:           "B77W",
		Name:           "Boeing 777-300ER",
		CruiseSpeed:    490,
		MaxSpeed:       525,
		MinSpeed:       150,
		MaxAltitude:    43100,
		ClimbRate:      1500,
		DescentRate:    2000,
		TurnRate:       2.5,
		WakeTurbulence: "H",
		Category:       "JET",
	},
	"A359": {
		ICAO:           "A359",
		Name:           "Airbus A350-900",
		CruiseSpeed:    490,
		MaxSpeed:       525,
		MinSpeed:       145,
		MaxAltitude:    43100,
		ClimbRate:      1800,
		DescentRate:    2200,
		TurnRate:       2.5,
		WakeTurbulence: "H",
		Category:       "JET",
	},
	"B752": {
		ICAO:           "B752",
		Name:           "Boeing 757-200",
		CruiseSpeed:    460,
		MaxSpeed:       500,
		MinSpeed:       135,
		MaxAltitude:    42000,
		ClimbRate:      2200,
		DescentRate:    2800,
		TurnRate:       3.0,
		WakeTurbulence: "M",
		Category:       "JET",
	},
	"A21N": {
		ICAO:           "A21N",
		Name:           "Airbus A321neo",
		CruiseSpeed:    455,
		MaxSpeed:       495,
		MinSpeed:       142,
		MaxAltitude:    39800,
		ClimbRate:      1900,
		DescentRate:    2400,
		TurnRate:       3.0,
		WakeTurbulence: "M",
		Category:       "JET",
	},
	"E75L": {
		ICAO:           "E75L",
		Name:           "Embraer E175",
		CruiseSpeed:    430,
		MaxSpeed:       470,
		MinSpeed:       125,
		MaxAltitude:    41000,
		ClimbRate:      2400,
		DescentRate:    2800,
		TurnRate:       3.5,
		WakeTurbulence: "M",
		Category:       "JET",
	},
	"CRJ9": {
		ICAO:           "CRJ9",
		Name:           "Bombardier CRJ-900",
		CruiseSpeed:    440,
		MaxSpeed:       475,
		MinSpeed:       130,
		MaxAltitude:    41000,
		ClimbRate:      2500,
		DescentRate:    3000,
		TurnRate:       3.5,
		WakeTurbulence: "M",
		Category:       "JET",
	},
	"B748": {
		ICAO:           "B748",
		Name:           "Boeing 747-8",
		CruiseSpeed:    490,
		MaxSpeed:       525,
		MinSpeed:       160,
		MaxAltitude:    43100,
		ClimbRate:      1400,
		DescentRate:    1800,
		TurnRate:       2.0,
		WakeTurbulence: "J",
		Category:       "JET",
	},
	"A388": {
		ICAO:           "A388",
		Name:           "Airbus A380-800",
		CruiseSpeed:    485,
		MaxSpeed:       520,
		MinSpeed:       165,
		MaxAltitude:    43100,
		ClimbRate:      1200,
		DescentRate:    1600,
		TurnRate:       1.8,
		WakeTurbulence: "J",
		Category:       "JET",
	},
}

// GetType returns the aircraft type by ICAO code
func GetType(icao string) (Type, bool) {
	t, exists := AircraftTypes[icao]
	return t, exists
}
