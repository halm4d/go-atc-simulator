# ATC Simulator

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Ebiten](https://img.shields.io/badge/Ebiten-2.x-4B8BBE)](https://ebitengine.org/)
[![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS%20%7C%20Web-lightgrey)]()
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
[![GitHub Pages](https://img.shields.io/badge/demo-Play%20Online-blue)](https://halm4d.github.io/go-atc-simulator/)

A realistic Air Traffic Control simulator built in Go with Ebiten, featuring real airports and aircraft types.

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Controls](#controls)
- [How to Play](#how-to-play)
- [Display](#display)
- [Aircraft Types](#aircraft-types)
- [Project Structure](#project-structure)
- [Future Enhancements](#future-enhancements)
- [Requirements](#requirements)
- [License](#license)
- [Contributing](#contributing)

## Features

- **Realistic ATC Radar Display**: Minimalist dark theme inspired by real ATC software
- **Real Airports**: Budapest (LHBP) with accurate runway layouts
- **STAR/SID Routes**: Standard Terminal Arrival Routes and Standard Instrument Departures with waypoints
- **Drag-to-Waypoint**: Click and drag an aircraft to a waypoint or route label to issue commands instantly
- **Flight Strip Panel**: Toggleable side panel showing aircraft list, details, and available commands
- **Real Aircraft Types**: 10 common aircraft types (B738, A320, B77W, A359, etc.) with realistic performance
- **Physics-Based Movement**: Aircraft follow realistic flight dynamics with wind effects
- **Separation Monitoring**: Enforces standard 5nm horizontal / 1000ft vertical separation
- **Command System**: Issue heading, altitude, speed, direct-to-waypoint, and holding commands
- **Holding Patterns**: Right-hand racetrack holding patterns
- **ILS Capture**: Automatic ILS establishment when aircraft are aligned with the runway
- **Runway Configuration**: Right-click near the airport to configure active landing/takeoff runways
- **Natural Language Commands**: Chat-based input with local LLM (Ollama) for natural ATC phrasing
- **Zoom & Pan**: Mouse wheel zoom from 2-60 pixels per nautical mile

## Quick Start

### Desktop (Native)

```bash
go run main.go
```

### Web Browser (WebAssembly)

**Windows:**
```bash
build-wasm.bat
cd web
python -m http.server 8080
```

**Linux/Mac:**
```bash
chmod +x build-wasm.sh
./build-wasm.sh
cd web
python3 -m http.server 8080
```

Then open your browser to `http://localhost:8080`

### Natural Language Commands (Optional)

The simulator supports natural language ATC commands via a local [Ollama](https://ollama.com/) instance. On startup, the game auto-detects Ollama and downloads the required model (`qwen2.5:0.5b`) automatically if needed.

**Setup:**

```bash
# Option 1: Install Ollama directly
# Download from https://ollama.com/ and run it

# Option 2: Use Docker
docker compose up -d
```

Once Ollama is running, start the game — it will connect automatically. If Ollama is not available, the game falls back to the built-in command parser.

## Controls

### Mouse
- **Click** on aircraft symbol or datatag to select (turns yellow)
- **Drag aircraft symbol** to a waypoint or route label to issue a direct-to or route assignment command
- **Drag datatag** to reposition it; right-click to reset
- **Drag waypoint/runway labels** to reposition; right-click to reset
- **Right-click near airport** to open runway configuration menu
- **Mouse Wheel** - Zoom in/out (over radar), scroll aircraft list (over flight strip panel), or adjust selected field value (when a datatag field is active)

### Keyboard
- **H** - Issue heading command (0-359 degrees)
- **A** - Issue altitude command (in hundreds of feet, e.g., 100 = 10,000 ft)
- **S** - Issue speed command (knots)
- **D** - Issue direct-to command (type waypoint name)
- **W** - Enter holding pattern
- **L** - Line up and wait (ground aircraft)
- **T** - Takeoff clearance (ground aircraft)
- **C** - Cleared to land (aircraft on final approach)
- **Tab** - Toggle flight strip panel
- **F1** - Toggle help overlay
- **Numbers/Letters** - Enter command value
- **ENTER** - Confirm command
- **ESC** - Deselect aircraft or cancel command

## How to Play

1. Select an airport from the menu and start the game
2. Aircraft spawn as arrivals (following STARs) and departures (queued at the runway)
3. Select an aircraft by clicking on it, then issue commands via keyboard or drag it to a waypoint/route
4. Guide arrivals onto the ILS approach and clear them to land
5. Clear departures for takeoff and let them follow their SID
6. Maintain at least 5nm horizontal OR 1000ft vertical separation between aircraft
7. Score increases with successful landings (+50) and departures (+25), decreases with separation violations

## Display

- **Green triangles**: Commanded arrival aircraft
- **White triangles**: Uncommanded arrival aircraft
- **Sky blue triangles**: Departure aircraft
- **Yellow triangles**: Selected aircraft
- **Amber symbols**: Ground/holding aircraft
- **Orange**: Aircraft on final approach awaiting landing clearance
- **Data tags**: Callsign, type, altitude, speed, heading/route with trend arrows
- **Cyan triangles**: Waypoints (STAR/SID/IAF/FAF fixes)
- **Blue/purple lines**: STAR and SID route paths with labels
- **White circle with crosshair**: Airport location
- **Bright gray lines**: Runways with numbers
- **Orange/Red pulsing circles**: Separation conflicts (orange = warning, red = critical)
- **Concentric circles**: Range rings every 10 nautical miles
- **Position trails**: Fading blue dots showing recent aircraft path
- **Flight strip panel** (right side): Aircraft list sorted by priority with detail view

## Aircraft Types

The simulator includes these realistic aircraft types:

- **B738** - Boeing 737-800 (Medium jet)
- **A320** - Airbus A320 (Medium jet)
- **B77W** - Boeing 777-300ER (Heavy jet)
- **A359** - Airbus A350-900 (Heavy jet)
- **B752** - Boeing 757-200 (Medium jet)
- **A21N** - Airbus A321neo (Medium jet)
- **E75L** - Embraer E175 (Medium jet)
- **CRJ9** - Bombardier CRJ-900 (Medium jet)
- **B748** - Boeing 747-8 (Super heavy)
- **A388** - Airbus A380-800 (Super heavy)

Each aircraft type has realistic performance characteristics including cruise speed, climb/descent rates, turn rates, and wake turbulence categories.

## Project Structure

```
atc-sim/
├── main.go                          # Entry point
├── internal/
│   ├── aircraft/
│   │   ├── types.go                 # Aircraft type definitions
│   │   └── aircraft.go              # Aircraft entity and physics
│   ├── airport/
│   │   ├── airport.go               # Airport and runway data
│   │   ├── waypoint.go              # Waypoint lookup and math
│   │   └── route.go                 # STAR/SID route definitions
│   ├── atc/
│   │   ├── command.go               # ATC command system
│   │   └── separation.go            # Separation monitoring
│   ├── data/
│   │   ├── loader.go                # JSON data loading
│   │   └── airports/
│   │       └── LHBP.json            # Budapest airport data
│   ├── chat/
│   │   └── message.go               # Chat message types and history
│   ├── game/
│   │   ├── game.go                  # Main game loop and state
│   │   ├── chat_executor.go         # Executes parsed ATC commands
│   │   ├── constants.go             # Game constants
│   │   └── input.go                 # Input handling
│   ├── nlp/
│   │   ├── nlp.go                   # NLP engine (Tier 1 parser + Tier 2 LLM)
│   │   ├── parser.go                # Rule-based ATC command parser
│   │   └── ollama.go                # Ollama LLM client (async queries, auto-pull)
│   └── render/
│       ├── radar.go                 # Radar display rendering
│       ├── flightstrip.go           # Flight strip panel UI
│       ├── chatpanel.go             # Chat panel UI
│       ├── textbox.go               # Command textbox UI
│       └── runwaymenu.go            # Runway configuration menu
├── config.example.json              # Example configuration
├── docker-compose.yml               # Ollama Docker setup
├── web/
│   ├── index.html                   # WASM loader page
│   ├── game.wasm                    # (generated)
│   └── wasm_exec.js                 # (copied from Go)
├── build-wasm.bat                   # Windows build script
└── build-wasm.sh                    # Linux/Mac build script
```

## Future Enhancements

- Additional airports (KJFK, KLAX, EGLL, etc.)
- Weather conditions
- Voice commands (speech-to-text)
- Multiplayer mode
- Mobile app version

## Requirements

- Go 1.21 or higher
- Modern web browser (for WASM version)

## License

MIT License

## Contributing

Contributions welcome! Feel free to open issues or submit pull requests.
