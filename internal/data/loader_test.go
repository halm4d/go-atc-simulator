package data

import (
	"atc-sim/internal/aircraft"
	"atc-sim/internal/airport"
	"testing"
)

func TestAircraftTypesLoaded(t *testing.T) {
	if len(aircraft.AircraftTypes) != 10 {
		t.Errorf("expected 10 aircraft types, got %d", len(aircraft.AircraftTypes))
	}
	for _, icao := range []string{"B738", "A320", "B77W", "A359", "B752", "A21N", "E75L", "CRJ9", "B748", "A388"} {
		if _, ok := aircraft.GetType(icao); !ok {
			t.Errorf("aircraft type %s not found", icao)
		}
	}
}

func TestAirportsLoaded(t *testing.T) {
	list := GetAirportList()
	if len(list) != 3 {
		t.Errorf("expected 3 airports, got %d", len(list))
	}
	for _, icao := range []string{"KJFK", "KLAX", "LHBP"} {
		apt := airport.GetAirport(icao)
		if apt == nil {
			t.Fatalf("airport %s not found", icao)
		}
		if apt.ICAO != icao {
			t.Errorf("expected ICAO %s, got %s", icao, apt.ICAO)
		}
		if len(apt.Runways) == 0 {
			t.Errorf("airport %s has no runways", icao)
		}
	}
}

func TestDefaultRunways(t *testing.T) {
	landing, takeoff := GetDefaultRunways("LHBP")
	if landing != "13R" || takeoff != "13L" {
		t.Errorf("LHBP defaults: expected 13R/13L, got %s/%s", landing, takeoff)
	}
}

func TestWaypointsLoaded(t *testing.T) {
	wps := airport.GetWaypoints("LHBP")
	if len(wps) == 0 {
		t.Error("LHBP has no waypoints")
	}
}

func TestRoutesLoaded(t *testing.T) {
	routes := airport.GetRoutes("LHBP")
	if len(routes) == 0 {
		t.Error("LHBP has no routes")
	}
}

func TestSIDExitsDerived(t *testing.T) {
	exits := airport.GetSIDExits("LHBP")
	if len(exits) != 4 {
		t.Errorf("expected 4 SID exits for LHBP, got %d", len(exits))
	}
}
