# NLP ATC Command System — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add natural language ATC command parsing with a chat UI, pilot readbacks, and optional Ollama LLM fallback.

**Architecture:** Two-tier parser (rule-based + optional Ollama) produces `ParsedCommand` structs that map to existing `Issue*` functions. A draggable chat panel replaces or supplements keyboard shortcuts. Config system stores user preferences.

**Tech Stack:** Go 1.26, Ebiten v2.9.9, `net/http` for Ollama API, `encoding/json` for config persistence.

**Spec:** `docs/superpowers/specs/2026-03-18-nlp-atc-commands-design.md`

---

## File Structure

### New Files

| File | Responsibility |
|------|---------------|
| `internal/config/config.go` | Config struct, Load/Save, UserConfigDir resolution |
| `internal/config/config_test.go` | Config load/save/defaults tests |
| `internal/nlp/command.go` | `ParsedCommand` struct definition |
| `internal/nlp/parser.go` | Rule-based ATC phraseology parser |
| `internal/nlp/parser_test.go` | Parser unit tests (main test file) |
| `internal/nlp/ollama.go` | Async Ollama HTTP client with channel-based results |
| `internal/nlp/ollama_test.go` | Ollama client tests |
| `internal/nlp/nlp.go` | Orchestrator: Tier 1 → Tier 2 → error |
| `internal/chat/message.go` | Chat message model + history buffer |
| `internal/chat/message_test.go` | Message history tests |
| `internal/render/chatpanel.go` | Chat panel rendering (message list + input bar) |
| `internal/game/chat_executor.go` | Maps ParsedCommand → Issue* calls with phase validation |
| `internal/game/chat_executor_test.go` | Executor phase validation tests |
| `docker-compose.yml` | Ollama container for local testing |
| `config.example.json` | Example config (committed) |

### Modified Files

| File | Changes |
|------|---------|
| `internal/atc/command.go:24-31` | Add `CommandDirect` and `CommandHold` to enum; add cases to `GetCommandString` |
| `internal/aircraft/aircraft.go:26-69` | Add `HasRequestedDeparture`, `HasRequestedLanding`, `HasRequestedInstructions`, `PrevHasRoute` fields |
| `internal/game/game.go:23-74` | Add `ChatHistory`, `ChatPanel`, `Config`, `OllamaClient`, `InputMode` fields to Game struct |
| `internal/game/game.go:90-100` | Load config in `startGame`, init chat + NLP |
| `internal/game/game.go:605-701` | Add pilot request triggers and PrevHasRoute tracking in `Update()` |
| `internal/game/game.go:791` | Branch `handleInput()` based on input mode |
| `internal/game/game.go:1336-1405` | Render chat panel in `Draw()` |
| `main.go:10-22` | Load config before creating game, pass to Game |

---

## Task 1: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `config.example.json`

- [ ] **Step 1: Write failing test for default config**

```go
// internal/config/config_test.go
package config

import "testing"

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.InputMode != "keyboard" {
		t.Errorf("expected default InputMode 'keyboard', got %q", cfg.InputMode)
	}
	if cfg.Ollama.Enabled != false {
		t.Error("expected Ollama disabled by default")
	}
	if cfg.Ollama.Endpoint != "http://localhost:11434" {
		t.Errorf("unexpected default Ollama endpoint: %s", cfg.Ollama.Endpoint)
	}
	if cfg.Ollama.Model != "llama3.2:1b" {
		t.Errorf("unexpected default Ollama model: %s", cfg.Ollama.Model)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal/config && go test -v -run TestDefaultConfig`
Expected: FAIL — package doesn't exist yet

- [ ] **Step 3: Implement config package**

```go
// internal/config/config.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type OllamaConfig struct {
	Enabled  bool   `json:"enabled"`
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

type Config struct {
	InputMode string       `json:"inputMode"`
	Ollama    OllamaConfig `json:"ollama"`
}

func DefaultConfig() Config {
	return Config{
		InputMode: "keyboard",
		Ollama: OllamaConfig{
			Enabled:  false,
			Endpoint: "http://localhost:11434",
			Model:    "llama3.2:1b",
		},
	}
}

// configDir returns the path to the atc-sim config directory.
func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to working directory
		return ".", nil
	}
	return filepath.Join(dir, "atc-sim"), nil
}

// configPath returns the full path to config.json.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads config from disk, returning defaults if file doesn't exist.
func Load() Config {
	path, err := configPath()
	if err != nil {
		return DefaultConfig()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig()
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}
	return cfg
}

// Save writes config to disk.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

- [ ] **Step 4: Write test for Save/Load roundtrip**

```go
// Add to internal/config/config_test.go
func TestSaveLoadRoundtrip(t *testing.T) {
	// Use a temp dir to avoid polluting real config
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.InputMode = "chat"
	cfg.Ollama.Enabled = true

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded := DefaultConfig()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.InputMode != "chat" {
		t.Errorf("expected InputMode 'chat', got %q", loaded.InputMode)
	}
	if !loaded.Ollama.Enabled {
		t.Error("expected Ollama enabled after load")
	}
}
```

- [ ] **Step 5: Run all config tests**

Run: `cd internal/config && go test -v`
Expected: PASS

- [ ] **Step 6: Create config.example.json**

```json
{
  "inputMode": "keyboard",
  "ollama": {
    "enabled": false,
    "endpoint": "http://localhost:11434",
    "model": "llama3.2:1b"
  }
}
```

- [ ] **Step 7: Commit**

```bash
git add internal/config/ config.example.json
git commit -m "feat: add config package with load/save and defaults"
```

---

## Task 2: ParsedCommand Struct + Command Enum Extension

**Files:**
- Create: `internal/nlp/command.go`
- Modify: `internal/atc/command.go:24-31` (add CommandDirect, CommandHold)
- Modify: `internal/atc/command.go:178-194` (add cases to GetCommandString)

- [ ] **Step 1: Create ParsedCommand struct**

```go
// internal/nlp/command.go
package nlp

// ParsedCommand is the output of the NLP parser (both rule-based and LLM).
// It is mapped to existing atc.Issue* calls by the game's chat executor.
type ParsedCommand struct {
	Callsign    string  // Matched aircraft callsign
	CommandType string  // "heading", "altitude", "speed", "direct", "takeoff", "lineup", "land", "hold"
	NumValue    float64 // For heading/altitude/speed
	StrValue    string  // For direct-to waypoint name
}
```

- [ ] **Step 2: Add CommandDirect and CommandHold to atc/command.go**

In `internal/atc/command.go`, change the const block at line 24-31:

```go
const (
	CommandHeading CommandType = iota
	CommandAltitude
	CommandSpeed
	CommandLineUpWait
	CommandClearedTakeoff
	CommandClearedLand
	CommandDirect
	CommandHold
)
```

- [ ] **Step 3: Add GetCommandString cases for new types**

In `internal/atc/command.go`, add cases before the closing `}` of `GetCommandString` (after line 191):

```go
	case CommandDirect:
		return fmt.Sprintf("%s DCT %s", c.Aircraft.Callsign, c.Aircraft.DirectTarget)
	case CommandHold:
		return fmt.Sprintf("%s HOLD", c.Aircraft.Callsign)
```

- [ ] **Step 4: Verify existing tests still pass**

Run: `go test ./internal/atc/ ./internal/aircraft/ ./internal/data/`
Expected: PASS (no existing behavior changed)

- [ ] **Step 5: Commit**

```bash
git add internal/nlp/command.go internal/atc/command.go
git commit -m "feat: add ParsedCommand struct and CommandDirect/CommandHold enum values"
```

---

## Task 3: Rule-Based Parser

**Files:**
- Create: `internal/nlp/parser.go`
- Create: `internal/nlp/parser_test.go`

- [ ] **Step 1: Write failing tests for basic commands**

```go
// internal/nlp/parser_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd internal/nlp && go test -v`
Expected: FAIL — `Parse` function doesn't exist yet

- [ ] **Step 3: Implement the rule-based parser**

```go
// internal/nlp/parser.go
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
```

- [ ] **Step 4: Run parser tests**

Run: `cd internal/nlp && go test -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/nlp/parser.go internal/nlp/parser_test.go
git commit -m "feat: add rule-based ATC phraseology parser with tests"
```

---

## Task 4: Chat Message Model + History

**Files:**
- Create: `internal/chat/message.go`
- Create: `internal/chat/message_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/chat/message_test.go
package chat

import "testing"

func TestHistoryAdd(t *testing.T) {
	h := NewHistory(3)
	h.Add(NewMessage(MsgATC, "ATC", "test1"))
	h.Add(NewMessage(MsgPilotReadback, "WZZ123", "test2"))
	if len(h.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(h.Messages))
	}
}

func TestHistoryMaxSize(t *testing.T) {
	h := NewHistory(3)
	h.Add(NewMessage(MsgATC, "ATC", "msg1"))
	h.Add(NewMessage(MsgATC, "ATC", "msg2"))
	h.Add(NewMessage(MsgATC, "ATC", "msg3"))
	h.Add(NewMessage(MsgATC, "ATC", "msg4"))
	if len(h.Messages) != 3 {
		t.Errorf("expected 3 messages (max), got %d", len(h.Messages))
	}
	if h.Messages[0].Text != "msg2" {
		t.Errorf("expected oldest message 'msg2', got %q", h.Messages[0].Text)
	}
}

func TestMessageTypes(t *testing.T) {
	m := NewMessage(MsgPilotRequest, "RYR456", "ready for departure")
	if m.Type != MsgPilotRequest {
		t.Errorf("wrong type: %d", m.Type)
	}
	if m.Sender != "RYR456" {
		t.Errorf("wrong sender: %s", m.Sender)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd internal/chat && go test -v`
Expected: FAIL — package doesn't exist

- [ ] **Step 3: Implement message model and history**

```go
// internal/chat/message.go
package chat

import "image/color"

// MessageType identifies the kind of chat message.
type MessageType int

const (
	MsgATC          MessageType = iota // Player's ATC command
	MsgPilotReadback                   // Pilot reading back a command
	MsgPilotRequest                    // Pilot-initiated request
	MsgSystem                          // System/error message
)

// Message is a single chat entry.
type Message struct {
	Type   MessageType
	Sender string // "ATC", callsign, or "SYS"
	Text   string
}

// NewMessage creates a chat message.
func NewMessage(t MessageType, sender, text string) Message {
	return Message{Type: t, Sender: sender, Text: text}
}

// Color returns the display color for this message type.
func (m Message) Color() color.RGBA {
	switch m.Type {
	case MsgATC:
		return color.RGBA{0, 200, 70, 255} // Green
	case MsgPilotReadback:
		return color.RGBA{0, 180, 220, 255} // Cyan
	case MsgPilotRequest:
		return color.RGBA{220, 160, 0, 255} // Amber
	case MsgSystem:
		return color.RGBA{255, 48, 48, 255} // Red
	}
	return color.RGBA{160, 165, 160, 255}
}

// Prefix returns the display prefix for this message.
func (m Message) Prefix() string {
	switch m.Type {
	case MsgATC:
		return "ATC"
	case MsgSystem:
		return "SYS"
	default:
		return m.Sender
	}
}

// History is a bounded FIFO buffer of chat messages.
type History struct {
	Messages []Message
	MaxSize  int
}

// NewHistory creates a history with the given capacity.
func NewHistory(maxSize int) *History {
	return &History{
		Messages: make([]Message, 0, maxSize),
		MaxSize:  maxSize,
	}
}

// Add appends a message, trimming the oldest if at capacity.
func (h *History) Add(m Message) {
	h.Messages = append(h.Messages, m)
	if len(h.Messages) > h.MaxSize {
		h.Messages = h.Messages[1:]
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd internal/chat && go test -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/chat/
git commit -m "feat: add chat message model and history buffer"
```

---

## Task 5: Chat Executor (ParsedCommand → Issue* Bridge)

**Files:**
- Create: `internal/game/chat_executor.go`
- Modify: `internal/aircraft/aircraft.go:26-69` (add pilot request flags)

- [ ] **Step 1: Add pilot request flags to Aircraft struct**

In `internal/aircraft/aircraft.go`, add these fields after the `AssignedRoute` field (line 68):

```go
	// Pilot request flags (for chat system)
	HasRequestedDeparture    bool
	HasRequestedLanding      bool
	HasRequestedInstructions bool
	PrevHasRoute             bool    // For detecting route-completion transitions
	HoldShortTimer           float64 // Seconds spent in PhaseHoldingShort
```

- [ ] **Step 2: Write the chat executor**

```go
// internal/game/chat_executor.go
package game

import (
	"atc-sim/internal/aircraft"
	"atc-sim/internal/atc"
	"atc-sim/internal/chat"
	"atc-sim/internal/nlp"
	"fmt"
	"math"
)

// executeParsedCommand maps a ParsedCommand to the appropriate Issue* function
// and returns a pilot readback message for the chat. On error, it returns a
// system error message.
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
// Uses the same formula as advanceRoute in aircraft.go for consistency.
func calculateHeadingTo(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1
	return airport.NormalizeHeading(90 - math.Atan2(dy, dx)*180/math.Pi)
}
```

Note: This file needs the `airport` import. Add to the import block:
```go
	"atc-sim/internal/airport"
	"strings"
```

- [ ] **Step 3: Verify project compiles**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add internal/game/chat_executor.go internal/aircraft/aircraft.go
git commit -m "feat: add chat executor and pilot request flags"
```

---

## Task 6: Ollama Async Client

**Files:**
- Create: `internal/nlp/ollama.go`
- Create: `internal/nlp/ollama_test.go`

- [ ] **Step 1: Write failing test for JSON response parsing**

```go
// internal/nlp/ollama_test.go
package nlp

import "testing"

func TestParseOllamaResponse_Heading(t *testing.T) {
	raw := `{"callsign": "WZZ123", "command": "heading", "value": 270}`
	cmd, err := parseOllamaJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Callsign != "WZZ123" || cmd.CommandType != "heading" || cmd.NumValue != 270 {
		t.Errorf("unexpected: %+v", cmd)
	}
}

func TestParseOllamaResponse_Direct(t *testing.T) {
	raw := `{"callsign": "RYR456", "command": "direct", "value": "ABONY"}`
	cmd, err := parseOllamaJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.CommandType != "direct" || cmd.StrValue != "ABONY" {
		t.Errorf("unexpected: %+v", cmd)
	}
}

func TestParseOllamaResponse_Error(t *testing.T) {
	raw := `{"error": "unrecognized"}`
	_, err := parseOllamaJSON(raw)
	if err == nil {
		t.Error("expected error for unrecognized response")
	}
}

func TestParseOllamaResponse_Takeoff(t *testing.T) {
	raw := `{"callsign": "WZZ123", "command": "takeoff", "value": null}`
	cmd, err := parseOllamaJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.CommandType != "takeoff" {
		t.Errorf("expected takeoff, got %s", cmd.CommandType)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd internal/nlp && go test -v -run TestParseOllama`
Expected: FAIL — function doesn't exist

- [ ] **Step 3: Implement Ollama client**

```go
// internal/nlp/ollama.go
package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OllamaResult is the outcome of an async LLM query.
type OllamaResult struct {
	Command *ParsedCommand
	Err     error
}

// OllamaClient handles async communication with a local Ollama instance.
type OllamaClient struct {
	Endpoint string
	Model    string
	ResultCh chan OllamaResult
}

// NewOllamaClient creates a client. ResultCh is buffered(1) so the goroutine never blocks.
func NewOllamaClient(endpoint, model string) *OllamaClient {
	return &OllamaClient{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Model:    model,
		ResultCh: make(chan OllamaResult, 1),
	}
}

// Ping tests connectivity by hitting the Ollama API root.
func (o *OllamaClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", o.Endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// QueryAsync sends the input to Ollama in a goroutine. Poll ResultCh for the answer.
func (o *OllamaClient) QueryAsync(input string, callsigns, waypoints []string) {
	go func() {
		cmd, err := o.query(input, callsigns, waypoints)
		o.ResultCh <- OllamaResult{Command: cmd, Err: err}
	}()
}

func (o *OllamaClient) query(input string, callsigns, waypoints []string) (*ParsedCommand, error) {
	systemPrompt := fmt.Sprintf(`You are an ATC command parser. Extract the structured command from this ATC instruction.
Active aircraft callsigns: %s
Active waypoints: %s

Respond ONLY with JSON in this format:
{"callsign": "XXX000", "command": "heading|altitude|speed|direct|takeoff|lineup|land|hold", "value": <number or string or null>}
If the input is not a valid ATC command, respond with: {"error": "unrecognized"}`,
		strings.Join(callsigns, ", "),
		strings.Join(waypoints, ", "))

	body := map[string]interface{}{
		"model":  o.Model,
		"system": systemPrompt,
		"prompt": input,
		"stream": false,
		"format": "json",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", o.Endpoint+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Ollama wraps the model output in {"response": "..."}
	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return parseOllamaJSON(ollamaResp.Response)
}

// parseOllamaJSON parses the model's JSON output into a ParsedCommand.
func parseOllamaJSON(raw string) (*ParsedCommand, error) {
	var result struct {
		Callsign string      `json:"callsign"`
		Command  string      `json:"command"`
		Value    interface{} `json:"value"`
		Error    string      `json:"error"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON from LLM: %w", err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("LLM: %s", result.Error)
	}
	if result.Callsign == "" || result.Command == "" {
		return nil, fmt.Errorf("incomplete response from LLM")
	}

	cmd := &ParsedCommand{
		Callsign:    strings.ToUpper(result.Callsign),
		CommandType: strings.ToLower(result.Command),
	}

	// Map value based on command type
	switch cmd.CommandType {
	case "heading", "altitude", "speed":
		switch v := result.Value.(type) {
		case float64:
			cmd.NumValue = v
		case string:
			// Try to parse string as number
			fmt.Sscanf(v, "%f", &cmd.NumValue)
		}
	case "direct":
		switch v := result.Value.(type) {
		case string:
			cmd.StrValue = strings.ToUpper(v)
		}
	}

	return cmd, nil
}
```

- [ ] **Step 4: Run Ollama tests**

Run: `cd internal/nlp && go test -v -run TestParseOllama`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/nlp/ollama.go internal/nlp/ollama_test.go
git commit -m "feat: add async Ollama client with JSON response parsing"
```

---

## Task 7: NLP Orchestrator

**Files:**
- Create: `internal/nlp/nlp.go`

- [ ] **Step 1: Implement the orchestrator**

```go
// internal/nlp/nlp.go
package nlp

// Engine is the NLP orchestrator. It tries the rule-based parser first,
// then optionally falls back to Ollama.
type Engine struct {
	Ollama *OllamaClient // nil if Ollama is disabled
}

// NewEngine creates an NLP engine. Pass nil for ollama to disable LLM fallback.
func NewEngine(ollama *OllamaClient) *Engine {
	return &Engine{Ollama: ollama}
}

// Process tries to parse the input. Returns a command synchronously if Tier 1
// succeeds. If Tier 1 fails and Ollama is available, it fires an async query
// and returns (nil, nil) — the caller should poll Ollama.ResultCh.
// If both tiers are unavailable, returns (nil, error).
func (e *Engine) Process(input string, callsigns, waypoints []string) (*ParsedCommand, error) {
	// Tier 1: rule-based parser
	cmd, err := Parse(input, callsigns)
	if err == nil {
		return cmd, nil
	}

	// Tier 2: Ollama fallback (async)
	if e.Ollama != nil {
		e.Ollama.QueryAsync(input, callsigns, waypoints)
		return nil, nil // caller polls ResultCh
	}

	return nil, err // No LLM available — return parser error
}
```

- [ ] **Step 2: Verify project compiles**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/nlp/nlp.go
git commit -m "feat: add NLP orchestrator with Tier 1/Tier 2 fallback"
```

---

## Task 8: Chat Panel Rendering

**Files:**
- Create: `internal/render/chatpanel.go`

- [ ] **Step 1: Implement chat panel renderer**

```go
// internal/render/chatpanel.go
package render

import (
	"atc-sim/internal/chat"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ChatPanel is the draggable chat UI overlay.
type ChatPanel struct {
	X, Y          int
	Width, Height int
	Focused       bool
	InputText     string
	ScrollOffset  int // Number of messages scrolled up from bottom

	// Drag state
	IsDragging  bool
	DragOffsetX int
	DragOffsetY int

	titleBarH  int
	inputBarH  int
	lineH      int
	bgColor    color.RGBA
	borderColor color.RGBA
}

// NewChatPanel creates a chat panel at the bottom of the screen.
func NewChatPanel(screenW, screenH int) *ChatPanel {
	w := screenW - 40
	h := 180
	return &ChatPanel{
		X:           20,
		Y:           screenH - h - 10,
		Width:       w,
		Height:      h,
		titleBarH:   18,
		inputBarH:   22,
		lineH:       14,
		bgColor:     color.RGBA{6, 8, 6, 220},
		borderColor: color.RGBA{0, 120, 0, 180},
	}
}

// UpdateLayout repositions the panel when the screen resizes.
func (cp *ChatPanel) UpdateLayout(screenW, screenH int) {
	if !cp.IsDragging {
		// Keep width matched to screen but don't reposition if user has dragged
		cp.Width = screenW - 40
	}
}

// HandleInput processes keyboard and mouse input for the chat panel.
// Returns the submitted text (non-empty) when user presses Enter, or "" otherwise.
func (cp *ChatPanel) HandleInput(mouseX, mouseY int) string {
	// Handle drag
	cp.handleDrag(mouseX, mouseY)

	// Focus on click inside input bar or '/' key
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		inInputBar := mouseX >= cp.X && mouseX <= cp.X+cp.Width &&
			mouseY >= cp.Y+cp.Height-cp.inputBarH && mouseY <= cp.Y+cp.Height
		if inInputBar {
			cp.Focused = true
		} else {
			// Check if click is outside the panel entirely
			if mouseX < cp.X || mouseX > cp.X+cp.Width || mouseY < cp.Y || mouseY > cp.Y+cp.Height {
				cp.Focused = false
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && !cp.Focused {
		cp.Focused = true
		return ""
	}

	if !cp.Focused {
		return ""
	}

	// ESC unfocuses
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		cp.Focused = false
		return ""
	}

	// Enter submits
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		text := strings.TrimSpace(cp.InputText)
		cp.InputText = ""
		cp.ScrollOffset = 0
		return text
	}

	// Backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(cp.InputText) > 0 {
		cp.InputText = cp.InputText[:len(cp.InputText)-1]
		return ""
	}

	// Collect typed characters
	cp.InputText += string(ebiten.AppendInputChars(nil))

	// Scroll with mouse wheel when hovering over message area
	if mouseX >= cp.X && mouseX <= cp.X+cp.Width && mouseY >= cp.Y && mouseY < cp.Y+cp.Height-cp.inputBarH {
		_, wy := ebiten.Wheel()
		if wy > 0 {
			cp.ScrollOffset++
		} else if wy < 0 && cp.ScrollOffset > 0 {
			cp.ScrollOffset--
		}
	}

	return ""
}

func (cp *ChatPanel) handleDrag(mouseX, mouseY int) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Check if click is in the title bar
		if mouseX >= cp.X && mouseX <= cp.X+cp.Width &&
			mouseY >= cp.Y && mouseY <= cp.Y+cp.titleBarH {
			cp.IsDragging = true
			cp.DragOffsetX = mouseX - cp.X
			cp.DragOffsetY = mouseY - cp.Y
		}
	}
	if cp.IsDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			cp.X = mouseX - cp.DragOffsetX
			cp.Y = mouseY - cp.DragOffsetY
		} else {
			cp.IsDragging = false
		}
	}
}

// Draw renders the chat panel.
func (cp *ChatPanel) Draw(screen *ebiten.Image, history *chat.History) {
	x := float32(cp.X)
	y := float32(cp.Y)
	w := float32(cp.Width)
	h := float32(cp.Height)

	// Background
	vector.FillRect(screen, x, y, w, h, cp.bgColor, false)
	vector.StrokeRect(screen, x, y, w, h, 1, cp.borderColor, false)

	// Title bar
	vector.FillRect(screen, x, y, w, float32(cp.titleBarH), color.RGBA{0, 60, 0, 200}, false)
	ebitenutil.DebugPrintAt(screen, "RADIO", cp.X+4, cp.Y+3)

	// Message area
	msgAreaTop := cp.Y + cp.titleBarH
	msgAreaBottom := cp.Y + cp.Height - cp.inputBarH
	visibleLines := (msgAreaBottom - msgAreaTop) / cp.lineH

	if history != nil && len(history.Messages) > 0 {
		msgs := history.Messages
		// Apply scroll offset
		end := len(msgs) - cp.ScrollOffset
		if end < 0 {
			end = 0
		}
		start := end - visibleLines
		if start < 0 {
			start = 0
		}

		drawY := msgAreaTop + 2
		for i := start; i < end; i++ {
			msg := msgs[i]
			line := msg.Prefix() + ": " + msg.Text
			// Truncate if too long
			maxChars := (cp.Width - 10) / 6 // ~6px per char at debug font size
			if len(line) > maxChars {
				line = line[:maxChars-3] + "..."
			}
			ebitenutil.DebugPrintAt(screen, line, cp.X+4, drawY)
			drawY += cp.lineH
		}
	}

	// Input bar
	inputY := float32(cp.Y + cp.Height - cp.inputBarH)
	vector.FillRect(screen, x, inputY, w, float32(cp.inputBarH), color.RGBA{3, 5, 3, 240}, false)
	vector.StrokeLine(screen, x, inputY, x+w, inputY, 1, cp.borderColor, false)

	prompt := "> " + cp.InputText
	if cp.Focused {
		// Blinking cursor
		prompt += "_"
	}
	ebitenutil.DebugPrintAt(screen, prompt, cp.X+4, int(inputY)+4)
}

// IsFocused returns whether the chat input bar has focus.
func (cp *ChatPanel) IsFocused() bool {
	return cp.Focused
}
```

- [ ] **Step 2: Verify project compiles**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/render/chatpanel.go
git commit -m "feat: add draggable chat panel renderer"
```

---

## Task 9: Wire Everything into the Game Loop

**Files:**
- Modify: `internal/game/game.go:23-74` (Game struct fields)
- Modify: `internal/game/game.go:90-100` (startGame init)
- Modify: `internal/game/game.go:605-701` (Update — pilot requests, Ollama polling, PrevHasRoute)
- Modify: `internal/game/game.go:791` (handleInput branching)
- Modify: `internal/game/game.go:1336-1405` (Draw — chat panel)
- Modify: `main.go:10-22` (load config)

- [ ] **Step 1: Add new fields to Game struct**

In `internal/game/game.go`, add imports:
```go
	"atc-sim/internal/chat"
	"atc-sim/internal/config"
	"atc-sim/internal/nlp"
	"atc-sim/internal/render"
```

Add fields to the Game struct (after line 73, before the closing `}`):
```go
	// Chat / NLP system
	ChatHistory  *chat.History
	ChatPanel    *render.ChatPanel
	NLPEngine    *nlp.Engine
	OllamaClient *nlp.OllamaClient
	Config       config.Config
	PendingOllama bool // true when waiting for Ollama response
```

- [ ] **Step 2: Load config in main.go**

Replace `main.go` with:
```go
package main

import (
	"atc-sim/internal/config"
	"atc-sim/internal/game"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowTitle("ATC Simulator")
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	cfg := config.Load()
	g := game.NewGameWithConfig(cfg)

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
```

- [ ] **Step 3: Add NewGameWithConfig constructor**

In `internal/game/game.go`, add after the existing `NewGame` function:
```go
// NewGameWithConfig creates a new game with the given configuration.
func NewGameWithConfig(cfg config.Config) *Game {
	g := NewGame()
	g.Config = cfg
	return g
}
```

- [ ] **Step 4: Initialize chat and NLP in startGame**

In `internal/game/game.go`, at the end of `startGame()` (before the closing `}`), add:

```go
	// Initialize chat system
	g.ChatHistory = chat.NewHistory(50)
	g.ChatPanel = render.NewChatPanel(g.Renderer.ScreenWidth, g.Renderer.ScreenHeight)

	// Initialize NLP engine
	if g.Config.Ollama.Enabled {
		g.OllamaClient = nlp.NewOllamaClient(g.Config.Ollama.Endpoint, g.Config.Ollama.Model)
		if err := g.OllamaClient.Ping(); err != nil {
			g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS", "LLM unavailable, using standard parser"))
			g.OllamaClient = nil
		} else {
			g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS",
				fmt.Sprintf("LLM connected (%s)", g.Config.Ollama.Model)))
		}
	}
	g.NLPEngine = nlp.NewEngine(g.OllamaClient)
```

- [ ] **Step 5: Add F2 toggle and chat input handling in handleInput**

In `internal/game/game.go`, at the very top of `handleInput()` (line 791), add before the existing `handleUIToggles()` call:

```go
	// F2 toggles input mode
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyF2) {
		if g.Config.InputMode == "chat" {
			g.Config.InputMode = "keyboard"
			if g.ChatPanel != nil {
				g.ChatPanel.Focused = false
			}
		} else {
			g.Config.InputMode = "chat"
			// Cancel any in-progress keyboard command
			g.CommandMode = ""
			g.CommandInput = ""
		}
	}

	// Chat mode input handling
	if g.Config.InputMode == "chat" && g.ChatPanel != nil {
		submitted := g.ChatPanel.HandleInput(g.InputHandler.MouseX, g.InputHandler.MouseY)
		if submitted != "" {
			g.processChatInput(submitted)
		}

		// Poll Ollama result channel
		if g.PendingOllama && g.OllamaClient != nil {
			select {
			case result := <-g.OllamaClient.ResultCh:
				g.PendingOllama = false
				if result.Err != nil {
					g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS", "Say again?"))
				} else {
					msg, err := g.executeParsedCommand(result.Command)
					if err != nil {
						g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS", err.Error()))
					} else {
						g.ChatHistory.Add(msg)
					}
				}
			default:
				// Still waiting
			}
		}

		// If chat panel is focused, skip ALL keyboard shortcuts (including F1, Tab, etc.)
		if g.ChatPanel.IsFocused() {
			return
		}
	}
```

**Important:** This block goes before the `g.handleUIToggles()` call so that the early `return` when chat is focused also prevents F1 (help), Tab (flight strip), and all other keyboard shortcuts from firing while typing.

- [ ] **Step 6: Add processChatInput method**

Add this method to `internal/game/game.go`:

```go
// processChatInput handles a submitted chat message.
func (g *Game) processChatInput(text string) {
	// Block input while waiting for Ollama response
	if g.PendingOllama {
		g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS", "Please wait..."))
		return
	}

	// Add the player's message to chat
	g.ChatHistory.Add(chat.NewMessage(chat.MsgATC, "ATC", text))

	// Get active callsigns and waypoints for the parser
	callsigns := make([]string, len(g.Aircraft))
	for i, a := range g.Aircraft {
		callsigns[i] = a.Callsign
	}
	waypoints := make([]string, len(g.Waypoints))
	for i, wp := range g.Waypoints {
		waypoints[i] = wp.Name
	}

	// Run through NLP engine
	cmd, err := g.NLPEngine.Process(text, callsigns, waypoints)
	if err != nil {
		g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS", "Say again?"))
		return
	}
	if cmd == nil {
		// Ollama query was fired async
		g.PendingOllama = true
		g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS", "Processing..."))
		return
	}

	// Execute the parsed command
	msg, err := g.executeParsedCommand(cmd)
	if err != nil {
		g.ChatHistory.Add(chat.NewMessage(chat.MsgSystem, "SYS", err.Error()))
		return
	}
	g.ChatHistory.Add(msg)
}
```

- [ ] **Step 7: Add pilot request triggers in Update()**

In `internal/game/game.go`, in the `Update()` method, after the aircraft update loop (after line 638 `a.Update(deltaTime)`), add a pilot request check:

```go
	// Pilot-initiated requests (chat mode only)
	if g.Config.InputMode == "chat" && g.ChatHistory != nil {
		for _, a := range g.Aircraft {
			// Track per-aircraft hold short time
			if a.Phase == aircraft.PhaseHoldingShort {
				a.HoldShortTimer += deltaTime
			}

			// Departure: ready for departure after holding short 10+ seconds
			if a.Phase == aircraft.PhaseHoldingShort && a.IsDeparture &&
				!a.HasRequestedDeparture && a.HoldShortTimer >= 10 {
				a.HasRequestedDeparture = true
				g.ChatHistory.Add(chat.NewMessage(chat.MsgPilotRequest, a.Callsign,
					fmt.Sprintf("Holding short runway %s, ready for departure", a.RunwayName)))
			}

			// Arrival: requesting landing clearance on final
			if a.Phase == aircraft.PhaseFinal && !a.HasRequestedLanding {
				a.HasRequestedLanding = true
				g.ChatHistory.Add(chat.NewMessage(chat.MsgPilotRequest, a.Callsign,
					fmt.Sprintf("Established on approach runway %s, requesting landing clearance", a.RunwayName)))
			}

			// Route completion: requesting further instructions
			if a.IsArrival && a.PrevHasRoute && !a.HasRoute && !a.HasRequestedInstructions {
				a.HasRequestedInstructions = true
				g.ChatHistory.Add(chat.NewMessage(chat.MsgPilotRequest, a.Callsign,
					"Requesting further instructions"))
			}

			// Update PrevHasRoute for next frame
			a.PrevHasRoute = a.HasRoute
		}
	}
```

- [ ] **Step 8: Draw chat panel in Draw()**

In `internal/game/game.go`, in the `Draw()` method, before the help overlay section (before line 1401 `// Draw help overlay`), add:

```go
	// Draw chat panel
	if g.Config.InputMode == "chat" && g.ChatPanel != nil && g.ChatHistory != nil {
		g.ChatPanel.Draw(screen, g.ChatHistory)
	}
```

- [ ] **Step 9: Verify project compiles and runs**

Run: `go build ./...`
Expected: SUCCESS

Run: `go run main.go` (manual visual test — verify chat panel appears when F2 is pressed)

- [ ] **Step 10: Commit**

```bash
git add internal/game/game.go main.go
git commit -m "feat: wire chat, NLP, and pilot requests into game loop"
```

---

## Task 10: Docker Compose + Config Example

**Files:**
- Create: `docker-compose.yml`

- [ ] **Step 1: Create docker-compose.yml**

```yaml
# docker-compose.yml
services:
  ollama:
    image: ollama/ollama
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama

volumes:
  ollama_data:
```

- [ ] **Step 2: Commit**

```bash
git add docker-compose.yml
git commit -m "feat: add docker-compose for local Ollama testing"
```

---

## Task 11: Full Integration Test

- [ ] **Step 1: Run all unit tests**

Run: `go test ./...`
Expected: ALL PASS

- [ ] **Step 2: Build for all targets**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 3: Manual smoke test**

1. Run `go run main.go`
2. Select LHBP, start game
3. Press **F2** — chat panel should appear at bottom
4. Type `WZZ` + a visible callsign + ` heading 270` and press Enter
5. Verify: ATC message appears in green, pilot readback in cyan, aircraft turns
6. Press **F2** again — chat panel disappears, keyboard shortcuts work
7. Test error cases: type a command for a ground aircraft to climb — should show red error

- [ ] **Step 4: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix: integration test fixes"
```
