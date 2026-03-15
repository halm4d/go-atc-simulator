# ATC Simulator

A realistic Air Traffic Control simulator built in Go with Ebiten, featuring real airports and aircraft types.

## Features

- **Realistic ATC Radar Display**: Minimalist dark theme inspired by real ATC software
- **Real Airports**: JFK (KJFK), LAX (KLAX), and Budapest (LHBP) with accurate runway layouts
- **STAR Approach Maps**: Budapest features realistic Standard Terminal Arrival Routes with waypoints
- **Approach Routes Visualization**: Color-coded STAR paths displayed on radar
- **Real Aircraft Types**: 10 common aircraft types (B738, A320, B77W, A359, etc.) with realistic performance
- **Physics-Based Movement**: Aircraft follow realistic flight dynamics
- **Separation Monitoring**: Enforces standard 5nm horizontal / 1000ft vertical separation
- **Command System**: Issue heading, altitude, speed, and direct-to-waypoint commands to aircraft
- **Zoom Functionality**: Mouse wheel zoom from 2-20 pixels per nautical mile

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

## Controls

- **Click** on aircraft or datatag to select (turns yellow)
- **Click datatag multiple times** to rotate position (top-right → top-left → bottom-left → bottom-right)
- **Drag command textbox** by clicking and dragging the title bar to move it anywhere on screen
- **Mouse Wheel** - Zoom in/out (2-20 pixels per nautical mile)
- **H** - Issue heading command (0-359 degrees)
- **A** - Issue altitude command (in hundreds of feet, e.g., 100 = 10,000 ft)
- **S** - Issue speed command (knots)
- **D** - Issue direct-to command (type waypoint name: TORAZ, CATUZ, ATICO, NICRA, etc.)
- **Numbers/Letters** - Enter command value
- **ENTER** - Confirm command
- **ESC** - Deselect aircraft or cancel command

## How to Play

1. Aircraft will spawn around the airport as arrivals and departures
2. Select an aircraft by clicking on it
3. Issue commands to maintain safe separation
4. Keep aircraft at least 5nm apart horizontally OR 1000ft apart vertically
5. Score decreases with separation violations

## Display

- **Green triangles**: Aircraft (pointing in their heading direction)
- **Yellow triangles**: Selected aircraft
- **Semi-transparent lines behind aircraft**: Heading indicators (15nm trail showing direction of travel)
- **Data tags**: Show callsign, aircraft type, altitude (hundreds of feet), speed (rotatable by clicking)
- **Radial lines from center**: Compass headings every 30° (cardinal directions N/E/S/W are brighter)
- **Heading labels**: 000, 030, 060... 330 marked around the radar
- **Cyan triangles**: IAF waypoints (Initial Approach Fixes like TORAZ, CATUZ, ATICO, NICRA)
- **White circle with crosshair**: Airport location
- **Bright gray lines**: Runways with numbers
- **Orange/Red lines**: Separation conflicts (orange = warning, red = critical)
- **Concentric circles**: Range rings every 10 nautical miles

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
│   │   └── airport.go               # Airport and runway data
│   ├── atc/
│   │   ├── command.go               # ATC command system
│   │   └── separation.go            # Separation monitoring
│   ├── game/
│   │   ├── game.go                  # Main game loop
│   │   └── input.go                 # Input handling
│   └── render/
│       └── radar.go                 # Radar display rendering
├── web/
│   ├── index.html                   # WASM loader page
│   ├── game.wasm                    # (generated)
│   └── wasm_exec.js                 # (copied from Go)
├── build-wasm.bat                   # Windows build script
└── build-wasm.sh                    # Linux/Mac build script
```

## Future Enhancements

- Additional airports (LAX, EGLL, etc.)
- More complex scenarios
- Weather conditions
- Voice commands (text-to-speech)
- Multiplayer mode
- Flight strips
- STAR/SID procedures
- Mobile app version

## Requirements

- Go 1.21 or higher
- Modern web browser (for WASM version)

## License

MIT License

## Contributing

Contributions welcome! Feel free to open issues or submit pull requests.
