# NLP ATC Command System — Design Spec

## Overview

Add a natural language command system to atc-sim that lets players issue ATC instructions by typing free-text phraseology (e.g., "WZZ123 turn heading 270") instead of using keyboard shortcuts. Pilots respond with readbacks and initiate key requests in a chat window.

## Goals

- Players can issue all existing command types via natural language in a chat interface
- Pilots read back commands and initiate contextual requests (ready for departure, requesting clearance)
- Works fully offline with a rule-based parser; optional Ollama LLM enhances flexibility
- Both input modes (keyboard shortcuts and chat) coexist, toggled via settings
- Distributed-friendly: game works out of the box without Ollama

## Non-Goals

- Voice recognition / text-to-speech
- Multiplayer / networked ATC
- AI that makes autonomous ATC decisions for the player

---

## Architecture

### Command Parsing Pipeline

Two-tier parsing converts free-text into a `ParsedCommand` struct (defined below), which is then executed by calling the appropriate existing `Issue*` functions in `atc/command.go`.

#### ParsedCommand Struct

```go
// internal/nlp/command.go
type ParsedCommand struct {
    Callsign    string  // Matched aircraft callsign
    CommandType string  // "heading", "altitude", "speed", "direct", "takeoff", "lineup", "land", "hold"
    NumValue    float64 // For heading/altitude/speed
    StrValue    string  // For direct-to waypoint name
}
```

This struct is the output of both parser tiers. The game loop maps it to the appropriate existing `Issue*` function call (see Execution section below).

#### Tier 1 — Rule-Based Parser (always available, <1ms)

Pattern matching on ATC phraseology. Extracts three parts from input:

1. **Callsign** — first token(s), matched against active aircraft list. Uses prefix matching: if the input starts with a string that uniquely matches exactly one active aircraft's callsign prefix, it is accepted. If zero or multiple aircraft match, the command fails with an appropriate error message ("unknown callsign" or "ambiguous callsign, be more specific").
2. **Command keyword** — mapped to command type via keyword table
3. **Value** — number (heading/altitude/speed) or waypoint name (for direct)

Keyword matching table:

| Keywords | CommandType | Value Expected |
|----------|------------|----------------|
| `turn`, `heading`, `hdg` | `"heading"` | Number (degrees) |
| `climb`, `descend`, `altitude`, `FL`, `flight level` | `"altitude"` | Number (feet or FL) |
| `speed`, `reduce`, `increase` | `"speed"` | Number (knots) |
| `direct`, `proceed` | `"direct"` | Waypoint name (string) |
| `cleared for takeoff`, `cleared takeoff`, `takeoff` | `"takeoff"` | None |
| `line up`, `line up and wait` | `"lineup"` | None |
| `cleared to land`, `cleared land`, `clear to land` | `"land"` | None |
| `hold` | `"hold"` | None |

Tolerances:
- Case-insensitive
- "FL180" parsed as altitude 18000
- "4000" after climb/descend parsed as feet
- Missing runway numbers accepted (uses active runway)

#### Tier 2 — LLM Fallback (optional, via Ollama)

When Tier 1 cannot parse the input, it is sent to Ollama with a system prompt:

```
You are an ATC command parser. Extract the structured command from this ATC instruction.
Active aircraft callsigns: {list of active callsigns}
Active waypoints: {list of waypoint names}

Respond ONLY with JSON in this format:
{"callsign": "XXX000", "command": "heading|altitude|speed|direct|takeoff|lineup|land|hold", "value": <number or string or null>}
If the input is not a valid ATC command, respond with: {"error": "unrecognized"}
```

The JSON response is mapped to a `ParsedCommand` struct.

**Async execution:** The Ollama call runs in a goroutine. While waiting for the LLM response, a pending message appears in chat: `"Processing..."`. The result is delivered via a channel polled in the game's `Update()` loop. A 5-second timeout applies — if exceeded, the call is cancelled and "Say again" is shown. This prevents frame drops on both desktop and WASM.

```go
// internal/nlp/ollama.go
type OllamaClient struct {
    Endpoint string
    Model    string
    ResultCh chan OllamaResult // polled by game loop
}

type OllamaResult struct {
    Command *ParsedCommand
    Err     error
}
```

**Fallback chain:** Rule parser → Ollama async (if enabled and reachable) → "Say again" error in chat.

#### Command Execution

The game loop maps `ParsedCommand` to existing functions. This is the bridge between the NLP output and the existing command system.

```go
// internal/game/chat_executor.go
func (g *Game) executeParsedCommand(cmd *nlp.ParsedCommand) (readback string, err error) {
    ac := g.findAircraftByCallsign(cmd.Callsign)
    if ac == nil {
        return "", fmt.Errorf("unknown callsign: %s", cmd.Callsign)
    }

    switch cmd.CommandType {
    case "heading":
        // Validate: aircraft must be airborne
        if ac.Phase < aircraft.PhaseClimbout {
            return "", fmt.Errorf("%s is on the ground", cmd.Callsign)
        }
        return atc.IssueHeadingCommand(ac, cmd.NumValue, g.CommandHistory, g.SimTime), nil

    case "altitude":
        if ac.Phase < aircraft.PhaseClimbout {
            return "", fmt.Errorf("%s is on the ground", cmd.Callsign)
        }
        return atc.IssueAltitudeCommand(ac, cmd.NumValue, g.CommandHistory, g.SimTime), nil

    case "speed":
        if ac.Phase < aircraft.PhaseClimbout {
            return "", fmt.Errorf("%s is on the ground", cmd.Callsign)
        }
        return atc.IssueSpeedCommand(ac, cmd.NumValue, g.CommandHistory, g.SimTime), nil

    case "direct":
        // Resolve waypoint name to coordinates
        wp := g.findWaypointByName(cmd.StrValue)
        if wp == nil {
            return "", fmt.Errorf("unknown waypoint: %s", cmd.StrValue)
        }
        ac.DirectTarget = wp.Name
        ac.TargetHeading = calculateHeadingTo(ac.X, ac.Y, wp.X, wp.Y)
        ac.HasRoute = false
        g.CommandHistory.Add(atc.Command{Type: atc.CommandDirect, Aircraft: ac, Value: 0, Time: g.SimTime})
        return fmt.Sprintf("Direct %s, %s", wp.Name, ac.Callsign), nil

    case "takeoff":
        if ac.Phase != aircraft.PhaseLineUpWait && ac.Phase != aircraft.PhaseHoldingShort {
            return "", fmt.Errorf("%s is not in position for takeoff", cmd.Callsign)
        }
        return atc.IssueTakeoffClearance(ac, g.CommandHistory, g.SimTime), nil

    case "lineup":
        if ac.Phase != aircraft.PhaseHoldingShort {
            return "", fmt.Errorf("%s is not holding short", cmd.Callsign)
        }
        return atc.IssueLineUpWait(ac, g.CommandHistory, g.SimTime), nil

    case "land":
        if ac.Phase != aircraft.PhaseFinal {
            return "", fmt.Errorf("%s is not on final approach", cmd.Callsign)
        }
        return atc.IssueLandingClearance(ac, g.CommandHistory, g.SimTime), nil

    case "hold":
        if ac.Phase < aircraft.PhaseClimbout {
            return "", fmt.Errorf("%s is on the ground", cmd.Callsign)
        }
        ac.EnterHold()
        g.CommandHistory.Add(atc.Command{Type: atc.CommandHold, Aircraft: ac, Time: g.SimTime})
        return fmt.Sprintf("Holding, %s", ac.Callsign), nil
    }
    return "", fmt.Errorf("unknown command type: %s", cmd.CommandType)
}
```

**Phase validation errors** are shown in chat as system messages (red), e.g., `"WZZ123 is not in position for takeoff"`. This gives the player clear feedback on why a command was rejected.

### New Package: `internal/nlp/`

- `command.go` — `ParsedCommand` struct definition
- `parser.go` — Rule-based parser: `Parse(input string, aircraft []*aircraft.Aircraft) (*ParsedCommand, error)`
- `ollama.go` — Ollama async HTTP client: `Query(input string, callsigns []string, waypoints []string)`, results via channel
- `nlp.go` — Orchestrator: tries Tier 1, falls back to Tier 2 async, returns command or error

---

## Chat UI

### Layout

A draggable panel, default position at the bottom of the screen.

- **Message area** — scrollable list, newest messages at bottom, auto-scrolls. Max 50 messages in memory.
- **Input bar** — single-line text field at panel bottom. Enter to send, ESC to clear/unfocus.
- **Drag handle** — title bar for repositioning the panel anywhere on screen.

### Message Types and Colors

| Type | Prefix | Color | Example |
|------|--------|-------|---------|
| ATC command (player) | `ATC:` | Green (#00C846) | `ATC: WZZ123 turn heading 270` |
| Pilot readback | `{callsign}:` | Cyan (#00B4DC) | `WZZ123: Heading 270, WZZ123` |
| Pilot request | `{callsign}:` | Amber (#DCA000) | `WZZ123: Holding short 13L, ready for departure` |
| System/error | `SYS:` | Red (#FF3030) | `WZZ123 is not on final approach` |

### Behavior

- Chat panel only visible when input mode is set to `chat`
- Pressing `/` or clicking the input bar focuses it
- While input bar is focused, keyboard shortcuts (H, A, S, etc.) are disabled
- Player can scroll up through message history
- Panel renders above the radar, semi-transparent background

### New Package: `internal/chat/`

- `message.go` — Message struct (type, sender, text, timestamp, color)
- `history.go` — Message history buffer (add, scroll, trim to 50)

### Chat Panel Rendering

Chat panel rendering lives in `internal/render/` (following the existing pattern where `textbox.go` and `flightstrip.go` handle their own rendering). A new file `internal/render/chatpanel.go` handles drawing the chat panel, message list, and input bar. The `internal/chat/` package owns only data and state — no rendering.

---

## Pilot AI

### Readbacks

Generated from the return value of `executeParsedCommand()`. The existing `Issue*` functions return ATC-phrased strings (controller's transmission). For the chat, we generate a **separate pilot readback** in pilot-speaking style. This is done in `executeParsedCommand` — after calling the `Issue*` function, it constructs and returns the pilot readback string that goes to chat.

Pilot readback templates:

- Heading: `"Heading {value}, {callsign}"`
- Altitude (climb): `"Climbing FL{value/100}, {callsign}"` or `"Climbing {value} feet, {callsign}"`
- Altitude (descend): `"Descending FL{value/100}, {callsign}"` or `"Descending {value} feet, {callsign}"`
- Speed: `"Speed {value}, {callsign}"`
- Takeoff: `"Cleared for takeoff runway {runway}, {callsign}"`
- Landing: `"Cleared to land runway {runway}, {callsign}"`
- Line up: `"Line up and wait runway {runway}, {callsign}"`
- Direct: `"Direct {waypoint}, {callsign}"`
- Hold: `"Holding, {callsign}"`

The existing `Issue*` return values continue to feed the `CommandHistory` for the flight strip panel. The pilot readback is a new, separate string that feeds the chat. Both systems work independently.

### Pilot-Initiated Requests

Triggered by game state checks in `Update()`. Each fires once per aircraft per trigger (boolean flag prevents repeat).

| Trigger Condition | Message Template | Aircraft Flag |
|-------------------|-----------------|---------------|
| Departure in `PhaseHoldingShort` for 10+ seconds | `"{callsign}, holding short runway {rwy}, ready for departure"` | `HasRequestedDeparture` |
| Arrival within 15nm of runway, in `PhaseFinal`, no landing clearance | `"{callsign}, established on approach runway {rwy}, requesting landing clearance"` | `HasRequestedLanding` |
| Arrival aircraft where `HasRoute` transitions from true to false (last waypoint consumed by `advanceRoute`) | `"{callsign}, reaching {lastWaypoint}, requesting further instructions"` | `HasRequestedInstructions` |

**Detection mechanism for last-waypoint trigger:** Add a `PrevHasRoute bool` field to Aircraft. In the game's `Update()` loop, for each aircraft: first check `if ac.PrevHasRoute && !ac.HasRoute` to detect the transition, then update `ac.PrevHasRoute = ac.HasRoute` at the end of each frame. This detects the frame when `advanceRoute` consumed the last waypoint. Only applies to arrival aircraft (`PhaseArrival`).

**Departures:** Departures follow SID routes to their exit point and then leave the simulation area. No "requesting further instructions" trigger is needed for departures — they are autonomous after takeoff clearance.

### Optional LLM Variation

When Ollama is available, pilot request templates can be sent to the LLM for slight rephrasing to add variety. This is a nice-to-have — if the LLM call fails or is slow, the template is used as-is.

---

## Settings & Configuration

### Config File

Stored using `os.UserConfigDir()` at `{userConfigDir}/atc-sim/config.json`. Falls back to working directory if `UserConfigDir` is unavailable. On first run, creates the file with defaults.

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

### Settings Screen

Accessible from the main menu. Options:

- **Input Mode**: toggle between `keyboard` and `chat`
- **Ollama Enabled**: on/off toggle
- **Ollama Endpoint**: text input for URL
- **Ollama Model**: text input for model name

### In-Game Toggle

- **F2** toggles input mode between keyboard and chat without leaving the game
- On toggle: any in-progress keyboard command (non-empty `CommandMode` or `CommandInput`) is cancelled first, then the mode switches and the chat panel visibility updates immediately
- On toggle from chat to keyboard: the chat input bar is unfocused and any typed-but-unsent text is preserved (not lost)

### Startup Behavior

When Ollama is enabled, the game sends a test request on startup:
- Success: chat shows `"LLM connected ({model})"`
- Failure: silently disables LLM, chat shows `"LLM unavailable, using standard parser"`

### New Package: `internal/config/`

- `config.go` — Config struct, Load/Save functions, default values, `UserConfigDir` resolution

---

## Docker Compose

A `docker-compose.yml` at the project root for local Ollama testing:

```yaml
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

After `docker compose up`, the player runs `docker exec ollama ollama pull llama3.2:1b` to download the model. A note in the README explains this setup.

---

## Integration with Existing Code

### Modified Files

| File | Changes |
|------|---------|
| `internal/game/game.go` | Add `ChatHistory`, `Config`, `OllamaClient` fields to Game struct. Add pilot request trigger checks in `Update()`. |
| `internal/game/input.go` | Add `handleChatInput()` function for text entry, enter/send, ESC handling. Branch in `handleInput()` based on input mode. Poll Ollama result channel. |
| `internal/render/radar.go` | Call chat panel render function when chat mode is active. |
| `internal/aircraft/aircraft.go` | Add boolean flags: `HasRequestedDeparture`, `HasRequestedLanding`, `HasRequestedInstructions`, `PrevHasRoute`. |
| `internal/atc/command.go` | No changes to existing functions. Add `CommandDirect` and `CommandHold` constants to the enum for command history tracking (the Issue* functions for these are in `chat_executor.go`, not here). |
| `main.go` | Load config on startup, pass to Game. |

### New Files

| File | Purpose |
|------|---------|
| `internal/nlp/command.go` | `ParsedCommand` struct |
| `internal/nlp/parser.go` | Rule-based ATC command parser |
| `internal/nlp/ollama.go` | Ollama async HTTP client with channel-based results |
| `internal/nlp/nlp.go` | Parser orchestrator (Tier 1 → Tier 2 async → error) |
| `internal/chat/message.go` | Chat message model and history buffer |
| `internal/render/chatpanel.go` | Chat panel rendering (message list + input bar) |
| `internal/game/chat_executor.go` | Maps `ParsedCommand` to existing `Issue*` calls with phase validation |
| `internal/config/config.go` | Settings load/save with `UserConfigDir` |
| `docker-compose.yml` | Ollama for local testing |
| `config.example.json` | Example configuration (committed to repo) |

### Unchanged

- `internal/atc/separation.go` — No changes.
- `internal/airport/` — No changes.
- `internal/data/` — No changes.

### WASM Compatibility

- Rule-based parser: pure Go, works in WASM.
- Ollama client: uses `net/http` in a goroutine. In WASM, Go's `net/http` maps to browser `fetch()`. The async channel-based design avoids blocking the main thread.
- Chat UI: Ebiten rendering, works in WASM.
- Config: `os.UserConfigDir()` is unavailable in WASM. Fallback: use hardcoded defaults (no persistence). Config settings screen still works in-memory for the session.
