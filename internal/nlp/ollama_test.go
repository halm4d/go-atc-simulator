package nlp

import "testing"

func TestParseOllamaResponse_Heading(t *testing.T) {
	raw := `{"callsign": "WZZ123", "command": "heading", "value": 270}`
	cmd, _, err := parseOllamaJSON(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Callsign != "WZZ123" || cmd.CommandType != "heading" || cmd.NumValue != 270 {
		t.Errorf("unexpected: %+v", cmd)
	}
}

func TestParseOllamaResponse_Direct(t *testing.T) {
	raw := `{"callsign": "RYR456", "command": "direct", "value": "ABONY"}`
	cmd, _, err := parseOllamaJSON(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.CommandType != "direct" || cmd.StrValue != "ABONY" {
		t.Errorf("unexpected: %+v", cmd)
	}
}

func TestParseOllamaResponse_Error(t *testing.T) {
	raw := `{"error": "unrecognized"}`
	_, _, err := parseOllamaJSON(raw, nil)
	if err == nil {
		t.Error("expected error for unrecognized response")
	}
}

func TestParseOllamaResponse_Takeoff(t *testing.T) {
	raw := `{"callsign": "WZZ123", "command": "takeoff", "value": null}`
	cmd, _, err := parseOllamaJSON(raw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.CommandType != "takeoff" {
		t.Errorf("expected takeoff, got %s", cmd.CommandType)
	}
}
