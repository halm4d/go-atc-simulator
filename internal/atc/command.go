package atc

import (
	"atc-sim/internal/aircraft"
	"fmt"
)

// CommandType represents the type of ATC command
type CommandType int

const (
	CommandHeading CommandType = iota
	CommandAltitude
	CommandSpeed
	CommandLineUpWait
	CommandClearedTakeoff
	CommandClearedLand
)

// Command represents an ATC command
type Command struct {
	Type     CommandType
	Aircraft *aircraft.Aircraft
	Value    float64
	Time     float64 // Time when command was issued
}

// CommandHistory stores the history of commands issued
type CommandHistory struct {
	Commands []Command
	MaxSize  int
}

// NewCommandHistory creates a new command history
func NewCommandHistory(maxSize int) *CommandHistory {
	return &CommandHistory{
		Commands: make([]Command, 0, maxSize),
		MaxSize:  maxSize,
	}
}

// AddCommand adds a command to the history
func (ch *CommandHistory) AddCommand(cmd Command) {
	ch.Commands = append(ch.Commands, cmd)
	if len(ch.Commands) > ch.MaxSize {
		ch.Commands = ch.Commands[1:]
	}
}

// IssueHeadingCommand issues a heading command to an aircraft
func IssueHeadingCommand(a *aircraft.Aircraft, heading float64, history *CommandHistory, currentTime float64) string {
	a.Commanded = true
	// Exit holding pattern when a new heading is issued
	if a.Phase == aircraft.PhaseHolding {
		a.Phase = aircraft.PhaseArrival
	}
	a.CommandHeading(heading)
	if history != nil {
		history.AddCommand(Command{
			Type:     CommandHeading,
			Aircraft: a,
			Value:    heading,
			Time:     currentTime,
		})
	}
	return fmt.Sprintf("%s, turn heading %03d", a.Callsign, int(heading))
}

// IssueAltitudeCommand issues an altitude command to an aircraft
func IssueAltitudeCommand(a *aircraft.Aircraft, altitude float64, history *CommandHistory, currentTime float64) string {
	a.Commanded = true
	a.CommandAltitude(altitude)
	if history != nil {
		history.AddCommand(Command{
			Type:     CommandAltitude,
			Aircraft: a,
			Value:    altitude,
			Time:     currentTime,
		})
	}

	altStr := ""
	if altitude >= 18000 {
		altStr = fmt.Sprintf("flight level %d", int(altitude/100))
	} else {
		altStr = fmt.Sprintf("%d feet", int(altitude))
	}

	if altitude > a.Altitude {
		return fmt.Sprintf("%s, climb and maintain %s", a.Callsign, altStr)
	} else if altitude < a.Altitude {
		return fmt.Sprintf("%s, descend and maintain %s", a.Callsign, altStr)
	}
	return fmt.Sprintf("%s, maintain %s", a.Callsign, altStr)
}

// IssueSpeedCommand issues a speed command to an aircraft
func IssueSpeedCommand(a *aircraft.Aircraft, speed float64, history *CommandHistory, currentTime float64) string {
	a.Commanded = true
	a.CommandSpeed(speed)
	if history != nil {
		history.AddCommand(Command{
			Type:     CommandSpeed,
			Aircraft: a,
			Value:    speed,
			Time:     currentTime,
		})
	}

	if speed > a.Speed {
		return fmt.Sprintf("%s, increase speed to %d knots", a.Callsign, int(speed))
	} else if speed < a.Speed {
		return fmt.Sprintf("%s, reduce speed to %d knots", a.Callsign, int(speed))
	}
	return fmt.Sprintf("%s, maintain %d knots", a.Callsign, int(speed))
}

// IssueLineUpWait instructs a holding-short aircraft to line up and wait on the runway
func IssueLineUpWait(a *aircraft.Aircraft, history *CommandHistory, currentTime float64) string {
	a.Commanded = true
	a.Phase = aircraft.PhaseLineUpWait
	if history != nil {
		history.AddCommand(Command{
			Type:     CommandLineUpWait,
			Aircraft: a,
			Time:     currentTime,
		})
	}
	return fmt.Sprintf("%s, runway %s, line up and wait", a.Callsign, a.RunwayName)
}

// IssueTakeoffClearance clears an aircraft for takeoff
func IssueTakeoffClearance(a *aircraft.Aircraft, history *CommandHistory, currentTime float64) string {
	a.Commanded = true
	a.Phase = aircraft.PhaseTakeoffRoll
	if history != nil {
		history.AddCommand(Command{
			Type:     CommandClearedTakeoff,
			Aircraft: a,
			Time:     currentTime,
		})
	}
	return fmt.Sprintf("%s, runway %s, cleared for takeoff", a.Callsign, a.RunwayName)
}

// IssueLandingClearance clears an aircraft established on final to land
func IssueLandingClearance(a *aircraft.Aircraft, history *CommandHistory, currentTime float64) string {
	a.Commanded = true
	a.Phase = aircraft.PhaseLanding
	if history != nil {
		history.AddCommand(Command{
			Type:     CommandClearedLand,
			Aircraft: a,
			Time:     currentTime,
		})
	}
	return fmt.Sprintf("%s, runway %s, cleared to land", a.Callsign, a.RunwayName)
}

// GetCommandString returns a formatted string for the command
func (c *Command) GetCommandString() string {
	switch c.Type {
	case CommandHeading:
		return fmt.Sprintf("%s HDG %03d", c.Aircraft.Callsign, int(c.Value))
	case CommandAltitude:
		if c.Value >= 18000 {
			return fmt.Sprintf("%s ALT FL%d", c.Aircraft.Callsign, int(c.Value/100))
		}
		return fmt.Sprintf("%s ALT %d", c.Aircraft.Callsign, int(c.Value))
	case CommandSpeed:
		return fmt.Sprintf("%s SPD %d", c.Aircraft.Callsign, int(c.Value))
	case CommandLineUpWait:
		return fmt.Sprintf("%s L+W %s", c.Aircraft.Callsign, c.Aircraft.RunwayName)
	case CommandClearedTakeoff:
		return fmt.Sprintf("%s T/O %s", c.Aircraft.Callsign, c.Aircraft.RunwayName)
	case CommandClearedLand:
		return fmt.Sprintf("%s LND %s", c.Aircraft.Callsign, c.Aircraft.RunwayName)
	}
	return ""
}
