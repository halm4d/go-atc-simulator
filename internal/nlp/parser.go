package nlp

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Parse parses free-text ATC phraseology into a ParsedCommand.
// callsigns is the list of active aircraft callsigns for matching.
func Parse(input string, callsigns []string) (*ParsedCommand, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty input")
	}

	upper := strings.ToUpper(input)
	tokens := strings.Fields(upper)
	if len(tokens) < 2 {
		return nil, fmt.Errorf("input too short")
	}

	// Step 1: Match callsign (prefix match against active aircraft)
	callsign, restIndex, err := matchCallsign(tokens, callsigns)
	if err != nil {
		return nil, err
	}

	rest := strings.Join(tokens[restIndex:], " ")
	if rest == "" {
		return nil, fmt.Errorf("no command after callsign")
	}

	// Step 2: Match command type and extract value
	return parseCommand(callsign, rest)
}

// matchCallsign tries to match the first token(s) against active callsigns using prefix matching.
func matchCallsign(tokens []string, callsigns []string) (matched string, nextIndex int, err error) {
	// Try matching first token as a prefix
	prefix := tokens[0]
	var matches []string
	for _, cs := range callsigns {
		if strings.HasPrefix(strings.ToUpper(cs), prefix) {
			matches = append(matches, cs)
		}
	}

	if len(matches) == 1 {
		return matches[0], 1, nil
	}
	if len(matches) > 1 {
		return "", 0, fmt.Errorf("ambiguous callsign, be more specific")
	}
	return "", 0, fmt.Errorf("unknown callsign: %s", tokens[0])
}

var flRegex = regexp.MustCompile(`FL\s*(\d+)`)
var numberRegex = regexp.MustCompile(`(\d+)`)

func parseCommand(callsign, rest string) (*ParsedCommand, error) {
	cmd := &ParsedCommand{Callsign: callsign}

	// Multi-word patterns (checked first, longest match)
	switch {
	case matchPhrase(rest, "CLEARED FOR TAKEOFF") || matchPhrase(rest, "CLEARED TAKEOFF"):
		cmd.CommandType = "takeoff"
		return cmd, nil
	case matchPhrase(rest, "CLEARED TO LAND") || matchPhrase(rest, "CLEARED LAND") || matchPhrase(rest, "CLEAR TO LAND"):
		cmd.CommandType = "land"
		return cmd, nil
	case matchPhrase(rest, "LINE UP AND WAIT") || matchPhrase(rest, "LINE UP"):
		cmd.CommandType = "lineup"
		return cmd, nil
	}

	// Single-word keyword commands
	tokens := strings.Fields(rest)
	keyword := tokens[0]
	remaining := strings.Join(tokens[1:], " ")

	switch keyword {
	case "TAKEOFF":
		cmd.CommandType = "takeoff"
		return cmd, nil

	case "HOLD":
		cmd.CommandType = "hold"
		return cmd, nil

	case "TURN", "HEADING", "HDG":
		cmd.CommandType = "heading"
		val, err := extractNumber(remaining)
		if err != nil {
			// If keyword is TURN, check if next token is HEADING/HDG followed by number
			if keyword == "TURN" {
				innerTokens := strings.Fields(remaining)
				if len(innerTokens) >= 2 && (innerTokens[0] == "HEADING" || innerTokens[0] == "HDG") {
					val, err = extractNumber(strings.Join(innerTokens[1:], " "))
				}
			}
			if err != nil {
				return nil, fmt.Errorf("heading command requires a number")
			}
		}
		cmd.NumValue = val
		return cmd, nil

	case "CLIMB", "DESCEND", "ALTITUDE":
		cmd.CommandType = "altitude"
		val, err := extractAltitude(remaining)
		if err != nil {
			return nil, fmt.Errorf("altitude command requires a number or flight level")
		}
		cmd.NumValue = val
		return cmd, nil

	case "FL", "FLIGHT":
		cmd.CommandType = "altitude"
		if keyword == "FLIGHT" {
			// "FLIGHT LEVEL 180"
			remaining = strings.TrimPrefix(remaining, "LEVEL ")
			remaining = strings.TrimPrefix(remaining, "LEVEL")
		}
		val, err := extractNumber(remaining)
		if err != nil {
			return nil, fmt.Errorf("flight level requires a number")
		}
		cmd.NumValue = val * 100
		return cmd, nil

	case "SPEED", "REDUCE", "INCREASE":
		cmd.CommandType = "speed"
		val, err := extractNumber(remaining)
		if err != nil {
			// "REDUCE SPEED TO 210" / "INCREASE SPEED TO 250"
			trimmed := strings.TrimPrefix(remaining, "SPEED ")
			trimmed = strings.TrimPrefix(trimmed, "TO ")
			val, err = extractNumber(trimmed)
			if err != nil {
				return nil, fmt.Errorf("speed command requires a number")
			}
		}
		cmd.NumValue = val
		return cmd, nil

	case "DIRECT", "PROCEED":
		cmd.CommandType = "direct"
		// Next token is the waypoint name
		directTokens := strings.Fields(remaining)
		// Skip "TO" if present
		if len(directTokens) > 0 && directTokens[0] == "TO" {
			directTokens = directTokens[1:]
		}
		if len(directTokens) == 0 {
			return nil, fmt.Errorf("direct command requires a waypoint name")
		}
		cmd.StrValue = directTokens[0]
		return cmd, nil
	}

	return nil, fmt.Errorf("unrecognized command")
}

func matchPhrase(text, phrase string) bool {
	return strings.Contains(text, phrase)
}

func extractAltitude(s string) (float64, error) {
	// Check for FL pattern first: "FL180", "FL 180"
	if m := flRegex.FindStringSubmatch(s); len(m) > 1 {
		val, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, err
		}
		return val * 100, nil
	}

	// Strip filler words
	s = strings.NewReplacer(
		"AND MAINTAIN ", "",
		"TO ", "",
		"FEET", "",
	).Replace(s)

	return extractNumber(strings.TrimSpace(s))
}

func extractNumber(s string) (float64, error) {
	m := numberRegex.FindString(s)
	if m == "" {
		return 0, fmt.Errorf("no number found in %q", s)
	}
	return strconv.ParseFloat(m, 64)
}
