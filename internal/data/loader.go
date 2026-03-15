package data

import (
	"atc-sim/internal/aircraft"
	"atc-sim/internal/airport"
	"embed"
	"encoding/json"
	"io/fs"
	"sort"
	"strings"
)

//go:embed aircraft_types.json
var aircraftTypesJSON []byte

//go:embed airports
var airportsFS embed.FS

// airportData is the on-disk representation of an airport JSON file.
type airportData struct {
	airport.Airport
	DefaultLandingRunway string           `json:"defaultLandingRunway"`
	DefaultTakeoffRunway string           `json:"defaultTakeoffRunway"`
	Waypoints            []airport.Waypoint `json:"waypoints"`
	Routes               []airport.Route    `json:"routes"`
}

var (
	airportList    []string // sorted ICAO codes
	defaultRunways = map[string][2]string{} // icao -> [landing, takeoff]
)

func init() {
	loadAircraftTypes()
	loadAirports()
}

func loadAircraftTypes() {
	var types []aircraft.Type
	if err := json.Unmarshal(aircraftTypesJSON, &types); err != nil {
		panic("data: failed to load aircraft types: " + err.Error())
	}
	aircraft.RegisterTypes(types)
}

func loadAirports() {
	entries, err := fs.ReadDir(airportsFS, "airports")
	if err != nil {
		panic("data: failed to read airports directory: " + err.Error())
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		raw, err := fs.ReadFile(airportsFS, "airports/"+entry.Name())
		if err != nil {
			panic("data: failed to read airport file " + entry.Name() + ": " + err.Error())
		}

		var ad airportData
		if err := json.Unmarshal(raw, &ad); err != nil {
			panic("data: failed to parse airport file " + entry.Name() + ": " + err.Error())
		}

		apt := ad.Airport
		airport.RegisterAirport(&apt)
		airport.RegisterRoutes(apt.ICAO, ad.Routes)
		airport.RegisterWaypoints(apt.ICAO, ad.Waypoints)

		defaultRunways[apt.ICAO] = [2]string{ad.DefaultLandingRunway, ad.DefaultTakeoffRunway}
		airportList = append(airportList, apt.ICAO)
	}

	sort.Strings(airportList)
}

// GetAirportList returns the sorted ICAO codes of all loaded airports.
func GetAirportList() []string {
	return airportList
}

// GetDefaultRunways returns the default landing and takeoff runway for an airport.
func GetDefaultRunways(icao string) (landing, takeoff string) {
	rw := defaultRunways[icao]
	return rw[0], rw[1]
}
