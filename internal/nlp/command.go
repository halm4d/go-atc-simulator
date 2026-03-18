package nlp

// ParsedCommand is the output of the NLP parser (both rule-based and LLM).
type ParsedCommand struct {
	Callsign    string  // Matched aircraft callsign
	CommandType string  // "heading", "altitude", "speed", "direct", "takeoff", "lineup", "land", "hold"
	NumValue    float64 // For heading/altitude/speed
	StrValue    string  // For direct-to waypoint name
}
