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

Two-tier parsing converts free-text into existing `atc.Command` structs.

**Tier 1 — Rule-Based Parser** (always available, <1ms)

Pattern matching on ATC phraseology. Extracts three parts from input:

1. **Callsign** — first token(s), matched against active aircraft
2. **Command keyword** — mapped to command type via keyword table
3. **Value** — number (heading/altitude/speed) or waypoint name

Keyword matching table:

| Keywords | Command Type |
|----------|-------------|
| `turn`, `heading`, `hdg` | CommandHeading |
| `climb`, `descend`, `altitude`, `FL`, `flight level` | CommandAltitude |
| `speed`, `reduce`, `increase` | CommandSpeed |
| `direct`, `proceed` | CommandDirect |
| `cleared for takeoff`, `cleared takeoff`, `takeoff` | CommandClearedTakeoff |
| `line up`, `line up and wait` | CommandLineUpWait |
| `cleared to land`, `cleared land`, `clear to land` | CommandClearedLand |
| `hold` | CommandHold |

Tolerances:
- Case-insensitive
- "FL180" parsed as altitude 18000
- "4000" after climb/descend parsed as feet
- Missing runway numbers accepted (uses active runway)
- Callsign fuzzy match (e.g., "WZZ123" matches "WZZ123", partial matches attempted)

**Tier 2 — LLM Fallback** (optional, via Ollama)

When Tier 1 cannot parse the input, it is sent to Ollama with a system prompt:

```
You are an ATC command parser. Extract the structured command from this ATC instruction.
Respond ONLY with JSON in this format:
{"callsign": "XXX000", "command": "heading|altitude|speed|direct|takeoff|lineup|land|hold", "value": <number or string>}
If the input is not a valid ATC command, respond with: {"error": "unrecognized"}
```

The JSON response maps to the same `atc.Command` struct as Tier 1.

**Fallback chain:** Rule parser -> Ollama (if enabled and reachable) -> "Say again" error in chat.

### New Package: `internal/nlp/`

- `parser.go` — Rule-based parser: `Parse(input string, aircraft []*aircraft.Aircraft) (*atc.Command, error)`
- `ollama.go` — Ollama HTTP client: `Query(input string) (*atc.Command, error)`
- `nlp.go` — Orchestrator: tries Tier 1, falls back to Tier 2, returns command or error

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
| System/error | `SYS:` | Red (#FF3030) | `Say again, WZZ123?` |

### Behavior

- Chat panel only visible when input mode is set to `chat`
- Pressing `/` or clicking the input bar focuses it
- While input bar is focused, keyboard shortcuts (H, A, S, etc.) are disabled
- Player can scroll up through message history
- Panel renders above the radar, semi-transparent background

### New Package: `internal/chat/`

- `message.go` — Message struct (type, sender, text, timestamp, color)
- `history.go` — Message history buffer (add, scroll, trim to 50)
- `panel.go` — Chat panel UI state (position, size, focused, scroll offset, drag state)

---

## Pilot AI

### Readbacks

Generated immediately after a valid command is executed. Template-based, no LLM needed.

Templates (reusing existing readback format from `command.go`):

- Heading: `"Heading {value}, {callsign}"`
- Altitude (climb): `"Climbing FL{value/100}, {callsign}"` or `"Climbing {value} feet, {callsign}"`
- Altitude (descend): `"Descending FL{value/100}, {callsign}"` or `"Descending {value} feet, {callsign}"`
- Speed: `"Speed {value}, {callsign}"`
- Takeoff: `"Cleared for takeoff runway {runway}, {callsign}"`
- Landing: `"Cleared to land runway {runway}, {callsign}"`
- Line up: `"Line up and wait runway {runway}, {callsign}"`
- Direct: `"Direct {waypoint}, {callsign}"`
- Hold: `"Holding, {callsign}"`

### Pilot-Initiated Requests

Triggered by game state. Each fires once per aircraft per trigger (boolean flag prevents repeat).

| Trigger Condition | Message Template | Aircraft Flag |
|-------------------|-----------------|---------------|
| Departure holding short for 10+ seconds | `"{callsign}, holding short runway {rwy}, ready for departure"` | `HasRequestedDeparture` |
| Arrival within 15nm of runway, on final, no landing clearance | `"{callsign}, established on approach runway {rwy}, requesting landing clearance"` | `HasRequestedLanding` |
| Aircraft on STAR/route reaching last waypoint before airport | `"{callsign}, reaching {waypoint}, requesting further instructions"` | `HasRequestedInstructions` |

### Optional LLM Variation

When Ollama is available, pilot request templates can be sent to the LLM for slight rephrasing to add variety. This is a nice-to-have — if the LLM call fails or is slow, the template is used as-is.

---

## Settings & Configuration

### Config File

Stored as `config.json` in the game's working directory.

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
- On toggle, chat panel visibility updates immediately

### Startup Behavior

When Ollama is enabled, the game sends a test request on startup:
- Success: chat shows `"LLM connected ({model})"`
- Failure: silently disables LLM, chat shows `"LLM unavailable, using standard parser"`

### New Package: `internal/config/`

- `config.go` — Config struct, Load/Save functions, default values

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
| `internal/game/game.go` | Add `ChatPanel`, `Config` fields to Game struct. Branch in `handleInput()` based on input mode. Add pilot request triggers in `Update()`. |
| `internal/game/input.go` | Add `handleChatInput()` function for text entry, enter/send, ESC handling. |
| `internal/render/radar.go` | Render chat panel (message list + input bar). |
| `internal/aircraft/aircraft.go` | Add boolean flags: `HasRequestedDeparture`, `HasRequestedLanding`, `HasRequestedInstructions`. |
| `main.go` | Load config on startup, pass to Game. |

### New Files

| File | Purpose |
|------|---------|
| `internal/nlp/parser.go` | Rule-based ATC command parser |
| `internal/nlp/ollama.go` | Ollama HTTP client |
| `internal/nlp/nlp.go` | Parser orchestrator (Tier 1 -> Tier 2 -> error) |
| `internal/chat/message.go` | Chat message model |
| `internal/chat/history.go` | Message history buffer |
| `internal/chat/panel.go` | Chat panel UI state and rendering logic |
| `internal/config/config.go` | Settings load/save |
| `docker-compose.yml` | Ollama for local testing |
| `config.json` | Default configuration (gitignored, with `config.example.json` committed) |

### Unchanged

- `internal/atc/command.go` — Reused as-is. Both input modes produce the same command structs.
- `internal/atc/separation.go` — No changes.
- `internal/airport/` — No changes.
- `internal/data/` — No changes.

### WASM Compatibility

- Rule-based parser: pure Go, works in WASM.
- Ollama client: uses `net/http`, which works in WASM (browser fetch API).
- Chat UI: Ebiten rendering, works in WASM.
- Config: `os.ReadFile`/`os.WriteFile` — needs WASM fallback (localStorage or in-memory defaults). For initial implementation, WASM uses defaults only.
