package nlp

import "testing"

// helper to make a callsign list
func callsigns(names ...string) []string {
	return names
}

func TestParseHeadingCommand(t *testing.T) {
	tests := []struct {
		input   string
		heading float64
	}{
		{"WZZ123 turn heading 270", 270},
		{"WZZ123 heading 090", 90},
		{"WZZ123 hdg 180", 180},
	}
	for _, tt := range tests {
		cmd, err := Parse(tt.input, callsigns("WZZ123"))
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if cmd.Callsign != "WZZ123" {
			t.Errorf("Parse(%q): callsign = %q, want WZZ123", tt.input, cmd.Callsign)
		}
		if cmd.CommandType != "heading" {
			t.Errorf("Parse(%q): type = %q, want heading", tt.input, cmd.CommandType)
		}
		if cmd.NumValue != tt.heading {
			t.Errorf("Parse(%q): value = %v, want %v", tt.input, cmd.NumValue, tt.heading)
		}
	}
}

func TestParseAltitudeCommand(t *testing.T) {
	tests := []struct {
		input    string
		altitude float64
	}{
		{"WZZ123 climb FL180", 18000},
		{"WZZ123 descend 4000", 4000},
		{"WZZ123 climb and maintain FL350", 35000},
		{"WZZ123 altitude 5000", 5000},
	}
	for _, tt := range tests {
		cmd, err := Parse(tt.input, callsigns("WZZ123"))
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if cmd.CommandType != "altitude" {
			t.Errorf("Parse(%q): type = %q, want altitude", tt.input, cmd.CommandType)
		}
		if cmd.NumValue != tt.altitude {
			t.Errorf("Parse(%q): value = %v, want %v", tt.input, cmd.NumValue, tt.altitude)
		}
	}
}

func TestParseSpeedCommand(t *testing.T) {
	cmd, err := Parse("WZZ123 speed 210", callsigns("WZZ123"))
	if err != nil {
		t.Fatal(err)
	}
	if cmd.CommandType != "speed" || cmd.NumValue != 210 {
		t.Errorf("got type=%q value=%v, want speed/210", cmd.CommandType, cmd.NumValue)
	}
}

func TestParseDirectCommand(t *testing.T) {
	cmd, err := Parse("WZZ123 direct ABONY", callsigns("WZZ123"))
	if err != nil {
		t.Fatal(err)
	}
	if cmd.CommandType != "direct" || cmd.StrValue != "ABONY" {
		t.Errorf("got type=%q str=%q, want direct/ABONY", cmd.CommandType, cmd.StrValue)
	}
}

func TestParseTakeoffCommand(t *testing.T) {
	tests := []string{
		"WZZ123 cleared for takeoff",
		"WZZ123 cleared takeoff",
		"WZZ123 takeoff",
	}
	for _, input := range tests {
		cmd, err := Parse(input, callsigns("WZZ123"))
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", input, err)
			continue
		}
		if cmd.CommandType != "takeoff" {
			t.Errorf("Parse(%q): type = %q, want takeoff", input, cmd.CommandType)
		}
	}
}

func TestParseLineUpCommand(t *testing.T) {
	tests := []string{
		"WZZ123 line up and wait",
		"WZZ123 line up",
	}
	for _, input := range tests {
		cmd, err := Parse(input, callsigns("WZZ123"))
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", input, err)
			continue
		}
		if cmd.CommandType != "lineup" {
			t.Errorf("Parse(%q): type = %q, want lineup", input, cmd.CommandType)
		}
	}
}

func TestParseLandCommand(t *testing.T) {
	tests := []string{
		"WZZ123 cleared to land",
		"WZZ123 cleared land",
		"WZZ123 clear to land",
	}
	for _, input := range tests {
		cmd, err := Parse(input, callsigns("WZZ123"))
		if err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", input, err)
			continue
		}
		if cmd.CommandType != "land" {
			t.Errorf("Parse(%q): type = %q, want land", input, cmd.CommandType)
		}
	}
}

func TestParseHoldCommand(t *testing.T) {
	cmd, err := Parse("WZZ123 hold", callsigns("WZZ123"))
	if err != nil {
		t.Fatal(err)
	}
	if cmd.CommandType != "hold" {
		t.Errorf("got type=%q, want hold", cmd.CommandType)
	}
}

func TestParseCaseInsensitive(t *testing.T) {
	cmd, err := Parse("wzz123 HEADING 270", callsigns("WZZ123"))
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Callsign != "WZZ123" || cmd.CommandType != "heading" {
		t.Errorf("case insensitive failed: callsign=%q type=%q", cmd.Callsign, cmd.CommandType)
	}
}

func TestParseUnknownCallsign(t *testing.T) {
	_, err := Parse("FAKE99 heading 270", callsigns("WZZ123"))
	if err == nil {
		t.Error("expected error for unknown callsign")
	}
}

func TestParseAmbiguousCallsign(t *testing.T) {
	_, err := Parse("WZZ heading 270", callsigns("WZZ123", "WZZ456"))
	if err == nil {
		t.Error("expected error for ambiguous callsign")
	}
}

func TestParsePrefixCallsignUnique(t *testing.T) {
	cmd, err := Parse("WZZ1 heading 270", callsigns("WZZ123", "RYR456"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Callsign != "WZZ123" {
		t.Errorf("callsign = %q, want WZZ123", cmd.Callsign)
	}
}

func TestParseUnrecognizedCommand(t *testing.T) {
	_, err := Parse("WZZ123 do something weird", callsigns("WZZ123"))
	if err == nil {
		t.Error("expected error for unrecognized command")
	}
}
