// internal/game/chat_executor.go
package game

import (
	"atc-sim/internal/aircraft"
	"atc-sim/internal/airport"
	"atc-sim/internal/atc"
	"atc-sim/internal/chat"
	"atc-sim/internal/nlp"
	"fmt"
	"math"
	"strings"
)

// executeParsedCommand maps a ParsedCommand to the appropriate Issue* function
// and returns a pilot readback message for the chat.
func (g *Game) executeParsedCommand(cmd *nlp.ParsedCommand) (chat.Message, error) {
	ac := g.findAircraftByCallsign(cmd.Callsign)
	if ac == nil {
		return chat.Message{}, fmt.Errorf("unknown callsign: %s", cmd.Callsign)
	}

	var readback string
	var err error

	switch cmd.CommandType {
	case "heading":
		if ac.Phase < aircraft.PhaseClimbout {
			return chat.Message{}, fmt.Errorf("%s is on the ground", cmd.Callsign)
		}
		atc.IssueHeadingCommand(ac, cmd.NumValue, g.CommandHistory, g.SimTime)
		readback = fmt.Sprintf("Heading %03d, %s", int(cmd.NumValue), ac.Callsign)

	case "altitude":
		if ac.Phase < aircraft.PhaseClimbout {
			return chat.Message{}, fmt.Errorf("%s is on the ground", cmd.Callsign)
		}
		atc.IssueAltitudeCommand(ac, cmd.NumValue, g.CommandHistory, g.SimTime)
		if cmd.NumValue > ac.Altitude {
			readback = fmt.Sprintf("Climbing %s, %s", atc.FormatAltitude(int(cmd.NumValue)), ac.Callsign)
		} else {
			readback = fmt.Sprintf("Descending %s, %s", atc.FormatAltitude(int(cmd.NumValue)), ac.Callsign)
		}

	case "speed":
		if ac.Phase < aircraft.PhaseClimbout {
			return chat.Message{}, fmt.Errorf("%s is on the ground", cmd.Callsign)
		}
		atc.IssueSpeedCommand(ac, cmd.NumValue, g.CommandHistory, g.SimTime)
		readback = fmt.Sprintf("Speed %d, %s", int(cmd.NumValue), ac.Callsign)

	case "direct":
		if ac.Phase < aircraft.PhaseClimbout {
			return chat.Message{}, fmt.Errorf("%s is on the ground", cmd.Callsign)
		}
		wp := g.findWaypointByName(cmd.StrValue)
		if wp == nil {
			return chat.Message{}, fmt.Errorf("unknown waypoint: %s", cmd.StrValue)
		}
		ac.Commanded = true
		ac.DirectTarget = wp.Name
		ac.TargetHeading = calculateHeadingTo(ac.X, ac.Y, wp.X, wp.Y)
		ac.HasRoute = false
		ac.RouteWaypoints = nil
		ac.RouteNames = nil
		g.CommandHistory.AddCommand(atc.Command{Type: atc.CommandDirect, Aircraft: ac, Time: g.SimTime})
		readback = fmt.Sprintf("Direct %s, %s", wp.Name, ac.Callsign)

	case "takeoff":
		if ac.Phase != aircraft.PhaseLineUpWait && ac.Phase != aircraft.PhaseHoldingShort {
			return chat.Message{}, fmt.Errorf("%s is not in position for takeoff", cmd.Callsign)
		}
		atc.IssueTakeoffClearance(ac, g.CommandHistory, g.SimTime)
		readback = fmt.Sprintf("Cleared for takeoff runway %s, %s", ac.RunwayName, ac.Callsign)

	case "lineup":
		if ac.Phase != aircraft.PhaseHoldingShort {
			return chat.Message{}, fmt.Errorf("%s is not holding short", cmd.Callsign)
		}
		atc.IssueLineUpWait(ac, g.CommandHistory, g.SimTime)
		readback = fmt.Sprintf("Line up and wait runway %s, %s", ac.RunwayName, ac.Callsign)

	case "land":
		if ac.Phase != aircraft.PhaseFinal {
			return chat.Message{}, fmt.Errorf("%s is not on final approach", cmd.Callsign)
		}
		atc.IssueLandingClearance(ac, g.CommandHistory, g.SimTime)
		readback = fmt.Sprintf("Cleared to land runway %s, %s", ac.RunwayName, ac.Callsign)

	case "hold":
		if ac.Phase < aircraft.PhaseClimbout {
			return chat.Message{}, fmt.Errorf("%s is on the ground", cmd.Callsign)
		}
		ac.EnterHold()
		g.CommandHistory.AddCommand(atc.Command{Type: atc.CommandHold, Aircraft: ac, Time: g.SimTime})
		readback = fmt.Sprintf("Holding, %s", ac.Callsign)

	default:
		err = fmt.Errorf("unknown command type: %s", cmd.CommandType)
	}

	if err != nil {
		return chat.Message{}, err
	}
	return chat.NewMessage(chat.MsgPilotReadback, ac.Callsign, readback), nil
}

// findAircraftByCallsign finds an aircraft by exact callsign match.
func (g *Game) findAircraftByCallsign(callsign string) *aircraft.Aircraft {
	for _, a := range g.Aircraft {
		if a.Callsign == callsign {
			return a
		}
	}
	return nil
}

// findWaypointByName finds a waypoint by name (case-insensitive).
func (g *Game) findWaypointByName(name string) *airport.Waypoint {
	for i := range g.Waypoints {
		if strings.EqualFold(g.Waypoints[i].Name, name) {
			return &g.Waypoints[i]
		}
	}
	return nil
}

// calculateHeadingTo calculates the heading from (x1,y1) to (x2,y2) in degrees.
func calculateHeadingTo(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return airport.NormalizeHeading(90 - math.Atan2(dy, dx)*180/math.Pi)
}
