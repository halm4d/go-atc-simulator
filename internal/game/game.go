package game

import (
	"atc-sim/internal/aircraft"
	"atc-sim/internal/airport"
	"atc-sim/internal/atc"
	"atc-sim/internal/data"
	"atc-sim/internal/render"
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Game represents the main game state
type Game struct {
	Airport          *airport.Airport
	Waypoints        []airport.Waypoint
	Routes           []airport.Route
	Aircraft         []*aircraft.Aircraft
	SelectedAircraft *aircraft.Aircraft
	Renderer         *render.Renderer
	InputHandler     *InputHandler
	CommandHistory   *atc.CommandHistory
	Conflicts        []atc.Conflict
	CommandTextBox   *render.CommandTextBox
	RunwayMenu       *render.RunwayMenu
	FlightStripPanel *render.FlightStripPanel

	SimTime             float64
	LastUpdate          time.Time
	Score               int
	CommandMode         string // "", "HEADING", "ALTITUDE", "SPEED", "DIRECT"
	CommandInput        string // Accumulates numeric input
	ActiveLandingRunway string // Runway used for ILS capture and approach cone
	ActiveTakeoffRunway string // Runway used by departures
	ShowHelp            bool   // Help overlay visible

	// Spawning & difficulty
	SpawnTimer       float64 // Time since last arrival spawn
	SpawnInterval    float64 // Seconds between arrival spawns
	DepSpawnTimer    float64 // Time since last departure spawn
	DepSpawnInterval float64 // Seconds between departure spawns
	Difficulty       int     // Increases with landings
	LandedCount      int     // Total aircraft landed this session
	UsedCallsigns    map[string]bool
	GameOver         bool

	// Wind
	WindDirection float64 // Direction wind is FROM (degrees, 0=N, 90=E, ...)
	WindSpeed     float64 // Wind speed in knots
	WindChangeTimer float64 // Time accumulator for gradual wind changes

	// Airport selector
	State         string // "MENU" or "PLAYING"
	MenuAirports  []string
	MenuSelection int

	// Active data-tag field for scroll-to-change interaction
	ActiveField    string             // "ALTITUDE", "SPEED", "HEADING", or ""
	ActiveAircraft *aircraft.Aircraft // aircraft whose field is active

	// Drag aircraft symbol to waypoint/route
	SymbolDragCandidate *aircraft.Aircraft // set on mousedown over aircraft symbol
	SymbolDragStartX    int
	SymbolDragStartY    int
}

// NewGame creates a new game instance starting at the airport selector menu
func NewGame() *Game {
	game := &Game{
		Renderer:      render.NewRenderer(DefaultScreenWidth, DefaultScreenHeight, DefaultZoomLevel),
		InputHandler:  NewInputHandler(),
		State:         "MENU",
		MenuAirports:  data.GetAirportList(),
		MenuSelection: 0,
		LastUpdate:    time.Now(),
	}
	return game
}

// startGame initialises the simulation for the chosen airport
func (g *Game) startGame(icao string) {
	apt := airport.GetAirport(icao)

	// Default runways from data
	defaultLanding, defaultTakeoff := data.GetDefaultRunways(icao)

	g.Airport = apt
	g.Waypoints = airport.GetWaypoints(apt.ICAO)
	g.Routes = airport.GetRoutes(apt.ICAO)
	g.Aircraft = make([]*aircraft.Aircraft, 0)
	g.CommandHistory = atc.NewCommandHistory(CommandHistorySize)
	g.CommandTextBox = render.NewCommandTextBox(g.Renderer.ScreenWidth-330, 50, 300, 200)
	g.RunwayMenu = render.NewRunwayMenu()
	g.FlightStripPanel = render.NewFlightStripPanel(g.Renderer.ScreenWidth, g.Renderer.ScreenHeight)
	g.SimTime = 0
	g.LastUpdate = time.Now()
	g.Score = InitialScore
	g.CommandMode = ""
	g.CommandInput = ""
	g.ActiveLandingRunway = defaultLanding
	g.ActiveTakeoffRunway = defaultTakeoff
	g.SpawnInterval = SpawnIntervalArrival
	g.DepSpawnInterval = SpawnIntervalDeparture
	g.Difficulty = 1
	g.LandedCount = 0
	g.UsedCallsigns = make(map[string]bool)
	g.GameOver = false
	g.WindDirection = rand.Float64() * 360
	g.WindSpeed = InitialWindSpeedMin + rand.Float64()*(InitialWindSpeedMax-InitialWindSpeedMin)
	g.WindChangeTimer = 0
	g.State = "PLAYING"

	g.spawnInitialAircraft()
}

// spawnInitialAircraft spawns the initial set of aircraft
func (g *Game) spawnInitialAircraft() {
	// Spawn 3 arrivals (will use STARs if available)
	for i := 0; i < 3; i++ {
		g.spawnArrival()
	}

	// Departures — use the active takeoff runway plus a parallel if one exists
	depRunway1 := g.ActiveTakeoffRunway
	depRunway2 := depRunway1
	for _, rwy := range g.Airport.Runways {
		if rwy.Name1 != depRunway1 && rwy.Name2 != depRunway1 {
			depRunway2 = rwy.Name1
			break
		}
	}
	departures := []struct {
		callsign string
		typeCode string
		runway   string
	}{
		{"JBU202", "A21N", depRunway1},
		{"SWA505", "B738", depRunway2},
	}

	for _, dep := range departures {
		rwy, ok := g.Airport.FindRunwayByName(dep.runway)
		if !ok {
			continue
		}
		threshX, threshY, takeoffHdg := airport.GetRunwayThreshold(rwy, dep.runway)

		a := aircraft.NewAircraft(dep.callsign, dep.typeCode, threshX, threshY, g.Airport.Elevation, takeoffHdg, 0)
		a.IsDeparture = true
		a.Phase = aircraft.PhaseHoldingShort
		a.RunwayName = dep.runway
		a.RunwayHeading = takeoffHdg
		a.AirportElevation = g.Airport.Elevation
		a.TargetHeading = takeoffHdg
		g.assignSID(a)
		g.Aircraft = append(g.Aircraft, a)
		g.UsedCallsigns[dep.callsign] = true
	}
}

// airline prefixes and number ranges for random callsign generation
var airlinePrefixes = []string{
	"WZZ", "RYR", "LOT", "DLH", "AUA", "BAW", "AFR", "THY",
	"AAL", "DAL", "UAL", "SWA", "JBU", "CSA", "AFL", "SAS",
}

func getAircraftTypePool() []string {
	var pool []string
	for icao, t := range aircraft.AircraftTypes {
		// Exclude super-heavy aircraft (J wake turbulence) from random spawning
		if t.WakeTurbulence != "J" {
			pool = append(pool, icao)
		}
	}
	return pool
}

func randomAircraftType() string {
	pool := getAircraftTypePool()
	return pool[rand.Intn(len(pool))]
}

// generateCallsign returns a unique random callsign
func (g *Game) generateCallsign() string {
	for attempts := 0; attempts < 50; attempts++ {
		prefix := airlinePrefixes[rand.Intn(len(airlinePrefixes))]
		number := rand.Intn(900) + 100 // 100–999
		cs := fmt.Sprintf("%s%d", prefix, number)
		if !g.UsedCallsigns[cs] {
			g.UsedCallsigns[cs] = true
			return cs
		}
	}
	// Fallback with sim time to guarantee uniqueness
	return fmt.Sprintf("SIM%d", int(g.SimTime))
}

// isSafeToSpawn checks that a proposed spawn position has enough separation
// from all existing airborne aircraft (at least 8nm horizontal or 2000ft vertical).
func (g *Game) isSafeToSpawn(x, y, altitude float64) bool {
	for _, a := range g.Aircraft {
		if a.Phase == aircraft.PhaseHoldingShort || a.Phase == aircraft.PhaseLineUpWait ||
			a.Phase == aircraft.PhaseLanded {
			continue // skip ground aircraft
		}
		dx := a.X - x
		dy := a.Y - y
		dist := math.Sqrt(dx*dx + dy*dy)
		altSep := math.Abs(a.Altitude - altitude)
		if dist < SpawnSafetyDistanceNm && altSep < SpawnSafetyAltitudeFt {
			return false
		}
	}
	return true
}

// assignRoute resolves a route's waypoint names to coordinates and assigns
// the full route to the aircraft for sequential navigation.
func (g *Game) assignRoute(a *aircraft.Aircraft, route airport.Route) {
	var coords [][2]float64
	var names []string
	for _, wpName := range route.Waypoints {
		wp := airport.FindWaypoint(g.Waypoints, wpName)
		if wp == nil {
			continue
		}
		coords = append(coords, [2]float64{wp.X, wp.Y})
		names = append(names, wp.Name)
	}
	if len(coords) == 0 {
		return
	}
	a.RouteWaypoints = coords
	a.RouteNames = names
	a.HasRoute = true
	a.RouteName = route.Name
	a.DirectTarget = names[0]
	// Preserve original assignment (only set once — first SID/STAR assigned at spawn)
	if a.AssignedRoute == "" {
		a.AssignedRoute = route.Name
	}
}

// spawnArrival spawns a new arrival. If STAR routes exist for the current airport,
// the aircraft is placed near the first STAR waypoint and follows the route.
// Otherwise it spawns at a random position ~30 nm out.
func (g *Game) spawnArrival() {
	typeCode := randomAircraftType()
	callsign := g.generateCallsign()

	// Collect STAR routes for the current airport
	var stars []airport.Route
	for _, r := range g.Routes {
		if r.Type == "STAR" {
			stars = append(stars, r)
		}
	}

	if len(stars) > 0 {
		// Shuffle STARs so we try different ones if the first pick is too close
		shuffled := make([]airport.Route, len(stars))
		copy(shuffled, stars)
		rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

		for _, star := range shuffled {
			firstWP := airport.FindWaypoint(g.Waypoints, star.Waypoints[0])
			if firstWP == nil {
				continue
			}
			// Spawn near the first STAR waypoint with a small offset
			angle := rand.Float64() * 2 * math.Pi
			offset := STARSpawnOffsetMin + rand.Float64()*(STARSpawnOffsetMax-STARSpawnOffsetMin)
			x := firstWP.X + offset*math.Cos(angle)
			y := firstWP.Y + offset*math.Sin(angle)

			dist := math.Sqrt(x*x + y*y)
			altitude := math.Min(14000, math.Max(8000, dist*350))

			if !g.isSafeToSpawn(x, y, altitude) {
				continue // try another STAR
			}

			heading := airport.HeadingTo(x, y, firstWP.X, firstWP.Y)
			speed := 250.0 + rand.Float64()*50

			a := aircraft.NewAircraft(callsign, typeCode, x, y, altitude, heading, speed)
			a.IsArrival = true
			a.Phase = aircraft.PhaseArrival
			g.Aircraft = append(g.Aircraft, a)
			return
		}
		// All STARs too close — skip this spawn cycle
		return
	}

	// Fallback: random spawn (no STARs available)
	g.spawnArrivalRandom(callsign, typeCode)
}

// spawnArrivalRandom spawns an arrival at a random position ~30 nm from the airport.
// Retries up to 10 times to find a safe position with enough separation.
func (g *Game) spawnArrivalRandom(callsign, typeCode string) {
	for attempt := 0; attempt < 10; attempt++ {
		angle := rand.Float64() * 360
		dist := 28.0 + rand.Float64()*7.0
		angleRad := (90 - angle) * math.Pi / 180
		x := dist * math.Cos(angleRad)
		y := dist * math.Sin(angleRad)
		altitude := 6000 + rand.Float64()*8000

		if !g.isSafeToSpawn(x, y, altitude) {
			continue
		}

		toAirport := airport.HeadingTo(x, y, 0, 0)
		heading := airport.NormalizeHeading(toAirport + (rand.Float64()*40 - 20))

		speed := 220 + rand.Float64()*80
		a := aircraft.NewAircraft(callsign, typeCode, x, y, altitude, heading, speed)
		a.IsArrival = true
		a.Phase = aircraft.PhaseArrival
		g.Aircraft = append(g.Aircraft, a)
		return
	}
	// All attempts too close — skip this spawn
}

// assignSID assigns a random SID route to a departure aircraft.
func (g *Game) assignSID(a *aircraft.Aircraft) {
	var sids []airport.Route
	for _, r := range g.Routes {
		if r.Type == "SID" {
			sids = append(sids, r)
		}
	}
	if len(sids) == 0 {
		return
	}
	g.assignRoute(a, sids[rand.Intn(len(sids))])
}

// spawnDeparture spawns a new departure at the active takeoff runway if slots allow
func (g *Game) spawnDeparture() {
	// Count pending departures (ground phases)
	pending := 0
	for _, a := range g.Aircraft {
		if a.IsDeparture && (a.Phase == aircraft.PhaseHoldingShort || a.Phase == aircraft.PhaseLineUpWait) {
			pending++
		}
	}
	if pending >= MaxPendingDepartures {
		return // Don't queue more than MaxPendingDepartures departures at once
	}

	rwyName := g.ActiveTakeoffRunway
	rwy, ok := g.Airport.FindRunwayByName(rwyName)
	if !ok {
		return
	}
	threshX, threshY, takeoffHdg := airport.GetRunwayThreshold(rwy, rwyName)

	typeCode := randomAircraftType()
	callsign := g.generateCallsign()

	a := aircraft.NewAircraft(callsign, typeCode, threshX, threshY, g.Airport.Elevation, takeoffHdg, 0)
	a.IsDeparture = true
	a.Phase = aircraft.PhaseHoldingShort
	a.RunwayName = rwyName
	a.RunwayHeading = takeoffHdg
	a.AirportElevation = g.Airport.Elevation
	a.TargetHeading = takeoffHdg
	g.assignSID(a)
	g.Aircraft = append(g.Aircraft, a)
}

// updateDifficulty adjusts spawn intervals based on landings
func (g *Game) updateDifficulty() {
	newDiff := 1 + g.LandedCount/LandingsPerDifficulty
	if newDiff > g.Difficulty {
		g.Difficulty = newDiff
		// Tighten spawn intervals (floor: MinSpawnIntervalArrival arrivals, MinSpawnIntervalDep departures)
		g.SpawnInterval = math.Max(MinSpawnIntervalArrival, SpawnIntervalArrival-float64(g.Difficulty-1)*DifficultyIntervalStep)
		g.DepSpawnInterval = math.Max(MinSpawnIntervalDep, SpawnIntervalDeparture-float64(g.Difficulty-1)*DifficultyIntervalStep)
	}
}

// applyWind applies wind drift to all airborne aircraft
func (g *Game) applyWind(deltaTime float64) {
	if g.WindSpeed < 0.5 {
		return
	}
	// Wind is FROM WindDirection, so aircraft drift in the opposite direction
	fromRad := (90 - g.WindDirection) * math.Pi / 180
	driftX := -g.WindSpeed * math.Cos(fromRad) * deltaTime / 3600.0
	driftY := -g.WindSpeed * math.Sin(fromRad) * deltaTime / 3600.0

	for _, a := range g.Aircraft {
		switch a.Phase {
		case aircraft.PhaseHoldingShort, aircraft.PhaseLineUpWait,
			aircraft.PhaseTakeoffRoll, aircraft.PhaseLanded:
			// Ground aircraft not drifted
		default:
			a.X += driftX
			a.Y += driftY
		}
	}
}

// handleMenuInput handles keyboard/mouse input on the airport selector screen
func (g *Game) handleMenuInput() {
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyArrowUp) || g.InputHandler.IsKeyJustPressed(ebiten.KeyW) {
		g.MenuSelection--
		if g.MenuSelection < 0 {
			g.MenuSelection = len(g.MenuAirports) - 1
		}
	}
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyArrowDown) || g.InputHandler.IsKeyJustPressed(ebiten.KeyS) {
		g.MenuSelection++
		if g.MenuSelection >= len(g.MenuAirports) {
			g.MenuSelection = 0
		}
	}
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyEnter) || g.InputHandler.IsKeyJustPressed(ebiten.KeySpace) {
		g.startGame(g.MenuAirports[g.MenuSelection])
	}

	// Mouse click on airport rows
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := g.InputHandler.MouseX, g.InputHandler.MouseY
		sw := float32(g.Renderer.ScreenWidth)
		sh := float32(g.Renderer.ScreenHeight)
		panelW := float32(460)
		panelX := sw/2 - panelW/2
		panelY := sh/2 - float32(240)/2
		listStartY := int(panelY) + 44
		rowH := 42

		for i := range g.MenuAirports {
			rowY := listStartY + i*rowH
			if mx >= int(panelX)+4 && mx <= int(panelX+panelW)-4 &&
				my >= rowY-2 && my < rowY+rowH-4 {
				if g.MenuSelection == i {
					// Click on already-selected row starts the game
					g.startGame(g.MenuAirports[g.MenuSelection])
				} else {
					g.MenuSelection = i
				}
				break
			}
		}
	}
}

// checkAutoDescend lowers arrival aircraft to 3000ft AGL when aligned with
// the runway approach heading and within 15nm of the threshold.
func (g *Game) checkAutoDescend() {
	for _, a := range g.Aircraft {
		if a.Phase != aircraft.PhaseArrival || a.RunwayName == "" {
			continue
		}
		dx := a.X - a.ThresholdX
		dy := a.Y - a.ThresholdY
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > AutoDescendDistanceNm {
			continue
		}

		// Must be heading roughly toward the runway
		hdgDiff := math.Abs(a.Heading - a.RunwayHeading)
		if hdgDiff > 180 {
			hdgDiff = 360 - hdgDiff
		}
		if hdgDiff > ApproachHeadingToleranceDeg {
			continue
		}

		targetAlt := g.Airport.Elevation + AutoDescendTargetAltFt
		if a.TargetAltitude > targetAlt {
			a.TargetAltitude = targetAlt
		}
	}
}

// checkILSCapture auto-establishes arrival aircraft on ILS when properly aligned
func (g *Game) checkILSCapture() {
	for _, a := range g.Aircraft {
		if a.Phase != aircraft.PhaseArrival {
			continue
		}
		for i := range g.Airport.Runways {
			rwy := &g.Airport.Runways[i]
			for _, runwayName := range []string{rwy.Name1, rwy.Name2} {
				// Only capture onto the active landing runway
				if runwayName != g.ActiveLandingRunway {
					continue
				}
				threshX, threshY, approachHdg := airport.GetRunwayThreshold(rwy, runwayName)

				dx := a.X - threshX
				dy := a.Y - threshY
				dist := math.Sqrt(dx*dx + dy*dy)

				// Must be within ILSCaptureDistanceNm
				if dist > ILSCaptureDistanceNm {
					continue
				}

				// Must be at or below the glide slope altitude (+tolerance)
				if a.Altitude > g.Airport.Elevation+dist*GlideSlopeGradientFtPerNm+GlideSlopeAltToleranceFt {
					continue
				}

				// Must be within ApproachHeadingToleranceDeg of the approach heading
				hdgDiff := math.Abs(a.Heading - approachHdg)
				if hdgDiff > 180 {
					hdgDiff = 360 - hdgDiff
				}
				if hdgDiff > ApproachHeadingToleranceDeg {
					continue
				}

				// Must be on the approach side of the threshold (not past it)
				approachRad := (90 - approachHdg) * math.Pi / 180
				approachVecX := math.Cos(approachRad)
				approachVecY := math.Sin(approachRad)
				if dx*(-approachVecX)+dy*(-approachVecY) <= 0 {
					continue
				}

				// Capture — establish on ILS
				a.Phase = aircraft.PhaseFinal
				a.RunwayName = runwayName
				a.RunwayHeading = approachHdg
				a.AirportElevation = g.Airport.Elevation
				a.ThresholdX = threshX
				a.ThresholdY = threshY
				a.TargetHeading = approachHdg
				break
			}
			if a.Phase == aircraft.PhaseFinal {
				break
			}
		}
	}
}

// removeAircraft removes a single aircraft from the simulation, cleaning up selection
func (g *Game) removeAircraft(a *aircraft.Aircraft) {
	delete(g.UsedCallsigns, a.Callsign)
	if g.SelectedAircraft == a {
		g.SelectedAircraft = nil
		g.CommandMode = ""
		g.CommandInput = ""
	}
}

// removeLandedAircraft removes aircraft that have touched down and awards score
func (g *Game) removeLandedAircraft() {
	var remaining []*aircraft.Aircraft
	for _, a := range g.Aircraft {
		if a.Phase == aircraft.PhaseLanded {
			g.Score += ScoreLanding
			g.LandedCount++
			g.removeAircraft(a)
		} else {
			remaining = append(remaining, a)
		}
	}
	g.Aircraft = remaining
}

// removeExitedAircraft removes aircraft that have flown beyond the radar sector (>50nm).
// Departures award +25 points; arrivals that wander off gain nothing.
func (g *Game) removeExitedAircraft() {
	var remaining []*aircraft.Aircraft
	for _, a := range g.Aircraft {
		dist := math.Sqrt(a.X*a.X + a.Y*a.Y)
		exited := dist > ExitRadiusNm &&
			(a.Phase == aircraft.PhaseDeparture ||
				a.Phase == aircraft.PhaseClimbout ||
				a.Phase == aircraft.PhaseArrival ||
				a.Phase == aircraft.PhaseHolding)
		if exited {
			if a.IsDeparture {
				g.Score += ScoreDeparture
			}
			g.removeAircraft(a)
		} else {
			remaining = append(remaining, a)
		}
	}
	g.Aircraft = remaining
}

// Update updates the game state
func (g *Game) Update() error {
	g.InputHandler.Update()

	if g.State == "MENU" {
		g.handleMenuInput()
		return nil
	}

	if g.GameOver {
		// Only handle restart input on game over
		if g.InputHandler.IsKeyJustPressed(ebiten.KeyEnter) || g.InputHandler.IsKeyJustPressed(ebiten.KeyR) {
			g.State = "MENU"
			g.GameOver = false
			g.MenuSelection = 0
		}
		return nil
	}

	// Calculate delta time
	now := time.Now()
	deltaTime := now.Sub(g.LastUpdate).Seconds()
	g.LastUpdate = now
	g.SimTime += deltaTime

	// Handle zoom
	g.handleZoom()

	// Handle input
	g.handleInput()

	// Update all aircraft
	for _, a := range g.Aircraft {
		a.Update(deltaTime)
	}

	// Apply wind drift to all airborne aircraft
	g.applyWind(deltaTime)

	// Gradually shift wind direction and speed
	g.WindChangeTimer += deltaTime
	if g.WindChangeTimer >= WindChangeIntervalSec {
		g.WindChangeTimer = 0
		g.WindDirection = airport.NormalizeHeading(g.WindDirection + (rand.Float64()*2*WindDirChangeDeg - WindDirChangeDeg))
		g.WindSpeed += (rand.Float64()*2*WindSpeedChangeKts - WindSpeedChangeKts)
		g.WindSpeed = math.Max(WindSpeedMin, math.Min(WindSpeedMax, g.WindSpeed))
	}

	// Auto-descend arrivals approaching the runway
	g.checkAutoDescend()

	// Auto-capture aircraft onto ILS when aligned with a runway
	g.checkILSCapture()

	// Remove aircraft that have landed and award score
	g.removeLandedAircraft()

	// Remove aircraft that have exited the sector (>50nm from airport)
	g.removeExitedAircraft()

	// Check for separation violations
	g.Conflicts = atc.CheckSeparation(g.Aircraft)

	// Update score based on conflicts
	if len(g.Conflicts) > 0 {
		for _, c := range g.Conflicts {
			if c.Severity == "CRITICAL" {
				g.Score -= CriticalConflictPenalty
			} else {
				g.Score -= ConflictPenalty
			}
		}
		if g.Score < 0 {
			g.Score = 0
		}
	}

	// Game over when score reaches 0
	if g.Score == 0 {
		g.GameOver = true
		return nil
	}

	// Continuous spawning
	g.SpawnTimer += deltaTime
	if g.SpawnTimer >= g.SpawnInterval {
		g.SpawnTimer = 0
		g.spawnArrival()
		g.updateDifficulty()
	}

	g.DepSpawnTimer += deltaTime
	if g.DepSpawnTimer >= g.DepSpawnInterval {
		g.DepSpawnTimer = 0
		g.spawnDeparture()
	}

	return nil
}

// handleZoom handles mouse wheel zoom, or nudges an active data-tag field value.
func (g *Game) handleZoom() {
	_, wheelY := ebiten.Wheel()
	if wheelY == 0 {
		return
	}

	// If mouse is over the flight strip panel, scroll the panel instead of zooming
	if g.FlightStripPanel != nil && g.FlightStripPanel.Visible {
		if g.FlightStripPanel.IsMouseInPanel(g.InputHandler.MouseX, g.InputHandler.MouseY) {
			g.FlightStripPanel.HandleScroll(wheelY, len(g.Aircraft), g.SelectedAircraft != nil)
			return
		}
	}

	// If a data-tag field is active and the aircraft is commandable, scroll changes the field.
	if g.ActiveField != "" && g.ActiveAircraft != nil && g.ActiveAircraft.IsCommandable() {
		a := g.ActiveAircraft
		switch g.ActiveField {
		case "ALTITUDE":
			a.TargetAltitude += 100 * wheelY
			if a.TargetAltitude < 1000 {
				a.TargetAltitude = 1000
			}
			if a.TargetAltitude > a.Type.MaxAltitude {
				a.TargetAltitude = a.Type.MaxAltitude
			}
			a.Commanded = true
		case "SPEED":
			a.TargetSpeed += 10 * wheelY
			if a.TargetSpeed < a.Type.MinSpeed {
				a.TargetSpeed = a.Type.MinSpeed
			}
			if a.TargetSpeed > a.Type.MaxSpeed {
				a.TargetSpeed = a.Type.MaxSpeed
			}
			a.Commanded = true
		case "HEADING":
			a.TargetHeading = airport.NormalizeHeading(a.TargetHeading + 5*wheelY)
			a.DirectTarget = "" // manual heading overrides any named target
			a.Commanded = true
		}
		return
	}

	// Normal zoom
	if wheelY > 0 {
		g.Renderer.ZoomIn(1.1)
	} else {
		g.Renderer.ZoomOut(1.1)
	}
}

// selectAircraft selects a new aircraft, deselecting any currently selected one.
func (g *Game) selectAircraft(a *aircraft.Aircraft) {
	if g.SelectedAircraft != nil {
		g.SelectedAircraft.Selected = false
	}
	g.SelectedAircraft = a
	a.Selected = true
	g.CommandMode = ""
	g.CommandInput = ""
	g.ActiveField = ""
	g.ActiveAircraft = nil
}

// deselectAircraft clears the current selection and all related state.
func (g *Game) deselectAircraft() {
	if g.SelectedAircraft != nil {
		g.SelectedAircraft.Selected = false
	}
	g.SelectedAircraft = nil
	g.CommandMode = ""
	g.CommandInput = ""
	g.ActiveField = ""
	g.ActiveAircraft = nil
}

// resetDragState clears all drag-to-waypoint state.
func (g *Game) resetDragState() {
	g.SymbolDragCandidate = nil
	g.Renderer.DraggingToWP = nil
	g.Renderer.DragToWPActive = false
	g.Renderer.DropTarget = ""
}

// handleInput processes user input
func (g *Game) handleInput() {
	if g.handleUIToggles() {
		return
	}
	if g.handleDragOperations() {
		return
	}
	if g.handleDragToWaypoint() {
		return
	}
	if g.handleMouseSelection() {
		return
	}
	g.handleRightClick()
	g.handleRunwayMenu()
	g.handleCommandInput()
}

// handleUIToggles handles Tab toggle, F1 help, help overlay ESC.
// Returns true if input was consumed.
func (g *Game) handleUIToggles() bool {
	// Toggle flight strip panel
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyTab) {
		if g.FlightStripPanel != nil {
			g.FlightStripPanel.Visible = !g.FlightStripPanel.Visible
		}
		return true
	}

	// Toggle help overlay
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyF1) {
		g.ShowHelp = !g.ShowHelp
		return true
	}
	if g.ShowHelp {
		if g.InputHandler.IsKeyJustPressed(ebiten.KeyEscape) {
			g.ShowHelp = false
		}
		return true
	}

	// Handle flight strip panel clicks
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.FlightStripPanel != nil && g.FlightStripPanel.Visible {
		if g.FlightStripPanel.IsMouseInPanel(g.InputHandler.MouseX, g.InputHandler.MouseY) {
			clicked := g.FlightStripPanel.HandleClick(g.InputHandler.MouseX, g.InputHandler.MouseY, g.Aircraft, g.SelectedAircraft)
			if clicked != nil {
				// Select new (or toggle off if same)
				if g.SelectedAircraft == clicked {
					g.deselectAircraft()
				} else {
					g.selectAircraft(clicked)
				}
			}
			return true
		}
	}

	return false
}

// handleDragOperations handles textbox drag, label/tag drag initiation,
// data tag drag update, and label drag update.
// Returns true if input was consumed.
func (g *Game) handleDragOperations() bool {
	// Handle textbox dragging
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Check if clicking textbox title bar to start dragging
		if g.CommandTextBox.IsMouseInTitleBar(g.InputHandler.MouseX, g.InputHandler.MouseY) {
			g.CommandTextBox.StartDrag(g.InputHandler.MouseX, g.InputHandler.MouseY)
			return true // Don't process other clicks when starting drag
		}
	}

	// Update drag position
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.CommandTextBox.UpdateDrag(g.InputHandler.MouseX, g.InputHandler.MouseY, g.Renderer.ScreenWidth, g.Renderer.ScreenHeight)
	}

	// Stop dragging
	if g.CommandTextBox.IsDragging && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		g.CommandTextBox.StopDrag()
	}

	// Label/tag dragging: waypoint labels, runway labels, and aircraft data tags
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.Renderer.DraggingLabel == "" && g.Renderer.DraggingAircraft == nil {
		mx, my := g.InputHandler.MouseX, g.InputHandler.MouseY

		// Check data tags first (highest priority — drawn on top)
		for _, a := range g.Aircraft {
			sx, sy := g.Renderer.WorldToScreen(a.X, a.Y)
			x1, y1, x2, y2 := g.Renderer.GetDataTagBounds(a, float32(sx), float32(sy))
			if mx >= x1 && mx <= x2 && my >= y1 && my <= y2 {
				g.Renderer.DraggingAircraft = a
				g.Renderer.DragOffsetX = mx - (int(sx) + a.DataTagOffX)
				g.Renderer.DragOffsetY = my - (int(sy) + a.DataTagOffY)
				if g.SelectedAircraft == a {
					// Already selected — activate field for scroll-to-change
					field := g.Renderer.GetDataTagFieldAt(a,
						float32(sx), float32(sy),
						float32(mx), float32(my))
					if field != "" {
						g.ActiveField = field
						g.ActiveAircraft = a
					} else {
						g.ActiveField = ""
						g.ActiveAircraft = nil
					}
				} else {
					// Select the aircraft whose tag was clicked
					g.selectAircraft(a)
				}
				return true
			}
		}

		// Check waypoint and runway labels
		hit := g.Renderer.GetLabelAt(mx, my)
		if hit != "" {
			rect := g.Renderer.LabelRects[hit]
			g.Renderer.DraggingLabel = hit
			g.Renderer.DragOffsetX = mx - rect.X
			g.Renderer.DragOffsetY = my - rect.Y
			return true
		}
	}
	// Update data tag drag
	if g.Renderer.DraggingAircraft != nil {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			a := g.Renderer.DraggingAircraft
			sx, sy := g.Renderer.WorldToScreen(a.X, a.Y)
			a.DataTagOffX = g.InputHandler.MouseX - g.Renderer.DragOffsetX - int(sx)
			a.DataTagOffY = g.InputHandler.MouseY - g.Renderer.DragOffsetY - int(sy)
		} else {
			g.Renderer.DraggingAircraft = nil
		}
		return true
	}
	// Update label drag (waypoints and runways)
	if g.Renderer.DraggingLabel != "" {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			lx := g.InputHandler.MouseX - g.Renderer.DragOffsetX
			ly := g.InputHandler.MouseY - g.Renderer.DragOffsetY
			key := g.Renderer.DraggingLabel

			if strings.HasPrefix(key, "rwy:") {
				// Runway label — find threshold screen position
				rwyName := strings.TrimPrefix(key, "rwy:")
				rwy, ok := g.Airport.FindRunwayByName(rwyName)
				if ok {
					tx, ty, _ := airport.GetRunwayThreshold(rwy, rwyName)
					sx, sy := g.Renderer.WorldToScreen(tx, ty)
					g.Renderer.RunwayLabelOffsets[rwyName] = [2]int{lx - int(sx), ly - int(sy)}
				}
			} else {
				// Waypoint label
				for _, wp := range g.Waypoints {
					if wp.Name == key {
						sx, sy := g.Renderer.WorldToScreen(wp.X, wp.Y)
						g.Renderer.WaypointLabelOffsets[wp.Name] = [2]int{lx - int(sx), ly - int(sy)}
						break
					}
				}
			}
		} else {
			g.Renderer.DraggingLabel = ""
		}
		return true
	}

	return false
}

// handleDragToWaypoint handles drag-to-waypoint threshold check,
// drop target tracking, mouse-up release with drop detection or
// click-to-select fallback, and ESC cancel.
// Returns true if input was consumed.
func (g *Game) handleDragToWaypoint() bool {
	// Drag-to-waypoint: threshold check (each frame while mouse held)
	if g.SymbolDragCandidate != nil && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		dx := g.InputHandler.MouseX - g.SymbolDragStartX
		dy := g.InputHandler.MouseY - g.SymbolDragStartY
		if !g.Renderer.DragToWPActive && (dx*dx+dy*dy) > DragThresholdPxSq {
			g.Renderer.DraggingToWP = g.SymbolDragCandidate
			g.Renderer.DragToWPActive = true
			// Select the aircraft being dragged
			g.selectAircraft(g.SymbolDragCandidate)
		}
	}

	// Drag-to-waypoint: track drop target each frame
	if g.Renderer.DragToWPActive && g.Renderer.DraggingToWP != nil {
		mx, my := g.InputHandler.MouseX, g.InputHandler.MouseY
		g.Renderer.DropTarget = ""
		if routeName := g.Renderer.GetRouteLabelAt(mx, my); routeName != "" {
			g.Renderer.DropTarget = "route:" + routeName
		} else if wpName := g.Renderer.GetLabelAt(mx, my); wpName != "" && !strings.HasPrefix(wpName, "rwy:") {
			g.Renderer.DropTarget = wpName
		} else if wpName := g.Renderer.GetWaypointNear(mx, my, 15, g.Waypoints); wpName != "" {
			g.Renderer.DropTarget = wpName
		}
	}

	// Drag-to-waypoint: release (mouse-up)
	if g.SymbolDragCandidate != nil && !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if g.Renderer.DragToWPActive && g.Renderer.DraggingToWP != nil {
			a := g.Renderer.DraggingToWP
			target := g.Renderer.DropTarget
			if strings.HasPrefix(target, "route:") {
				routeName := strings.TrimPrefix(target, "route:")
				route := airport.FindRoute(g.Routes, routeName)
				if route != nil {
					g.assignRoute(a, *route)
					a.Commanded = false
				}
			} else if target != "" {
				wp := airport.FindWaypoint(g.Waypoints, target)
				if wp != nil {
					heading := airport.HeadingTo(a.X, a.Y, wp.X, wp.Y)
					atc.IssueHeadingCommand(a, heading, g.CommandHistory, g.SimTime)
					a.DirectTarget = target
				}
			}
		} else {
			// No drag occurred — perform normal click-to-select
			g.selectAircraft(g.SymbolDragCandidate)
		}
		// Reset drag state
		g.resetDragState()
		return true
	}

	// ESC to deselect and clear any active field
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyEscape) {
		g.deselectAircraft()
		// Also cancel any in-progress drag
		g.resetDragState()
		return true
	}

	return false
}

// handleMouseSelection handles left-click aircraft selection (datatag click
// or symbol click), includes textbox check.
// Returns true if input was consumed.
func (g *Game) handleMouseSelection() bool {
	// Mouse click to select aircraft or start drag-to-waypoint (only if not clicking in textbox)
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Don't select aircraft if clicking inside the textbox
		if g.CommandTextBox.IsMouseInTextBox(g.InputHandler.MouseX, g.InputHandler.MouseY) {
			return true
		}

		worldX, worldY := g.Renderer.ScreenToWorld(g.InputHandler.MouseX, g.InputHandler.MouseY)
		clicked, isDataTagClick := g.InputHandler.HandleMouseClick(g.Aircraft, worldX, worldY, g.Renderer)
		if clicked != nil {
			if isDataTagClick {
				// Datatag click — handle selection/field activation (no drag-to-waypoint)
				if clicked == g.SelectedAircraft {
					sx, sy := g.Renderer.WorldToScreen(clicked.X, clicked.Y)
					field := g.Renderer.GetDataTagFieldAt(clicked,
						float32(sx), float32(sy),
						float32(g.InputHandler.MouseX), float32(g.InputHandler.MouseY))
					if field != "" {
						g.ActiveField = field
						g.ActiveAircraft = clicked
					} else {
						g.ActiveField = ""
						g.ActiveAircraft = nil
					}
				} else {
					g.selectAircraft(clicked)
				}
			} else if clicked.IsCommandable() {
				// Symbol click on commandable aircraft — defer selection, start drag candidate
				g.SymbolDragCandidate = clicked
				g.SymbolDragStartX = g.InputHandler.MouseX
				g.SymbolDragStartY = g.InputHandler.MouseY
			} else {
				// Symbol click on non-commandable aircraft — just select
				g.selectAircraft(clicked)
			}
		}
	}

	return false
}

// handleRightClick handles right-click to reset datatag, reset label,
// or open the airport runway menu.
func (g *Game) handleRightClick() {
	// Right-click on label/tag → reset position
	// Right-click near airport → open runway config menu
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		mx, my := g.InputHandler.MouseX, g.InputHandler.MouseY

		// Check data tags — right-click resets to default position
		for _, a := range g.Aircraft {
			sx, sy := g.Renderer.WorldToScreen(a.X, a.Y)
			x1, y1, x2, y2 := g.Renderer.GetDataTagBounds(a, float32(sx), float32(sy))
			if mx >= x1 && mx <= x2 && my >= y1 && my <= y2 {
				a.ResetDataTag()
				if g.ActiveAircraft == a {
					g.ActiveField = ""
					g.ActiveAircraft = nil
				}
				return
			}
		}

		// Check waypoint/runway labels — right-click resets to auto-placement
		if hit := g.Renderer.GetLabelAt(mx, my); hit != "" {
			if strings.HasPrefix(hit, "rwy:") {
				delete(g.Renderer.RunwayLabelOffsets, strings.TrimPrefix(hit, "rwy:"))
			} else {
				delete(g.Renderer.WaypointLabelOffsets, hit)
			}
			return
		}

		// Airport runway menu
		airportSX, airportSY := g.Renderer.WorldToScreen(0, 0)
		dx := float64(mx) - airportSX
		dy := float64(my) - airportSY
		if math.Sqrt(dx*dx+dy*dy) < AirportClickRadius {
			g.RunwayMenu.Show(mx+10, my, g.Airport, g.Renderer.ScreenWidth, g.Renderer.ScreenHeight)
			return
		}
	}
}

// handleRunwayMenu handles left-click on runway menu buttons or closes the menu.
func (g *Game) handleRunwayMenu() {
	// Left-click: handle runway menu buttons or close menu
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := g.InputHandler.MouseX, g.InputHandler.MouseY
		if g.RunwayMenu.Visible {
			rwy, action := g.RunwayMenu.HandleClick(mx, my)
			switch action {
			case "LAND":
				g.ActiveLandingRunway = rwy
			case "TAKEOFF":
				g.ActiveTakeoffRunway = rwy
			case "CLOSE":
				g.RunwayMenu.Hide()
			default:
				if !g.RunwayMenu.IsMouseInside(mx, my) {
					g.RunwayMenu.Hide()
				}
			}
		}
	}
}

// handleCommandInput handles command mode selection (H/A/S/D/W/L/T/C keys),
// number input, enter to confirm, and backspace.
func (g *Game) handleCommandInput() {
	// Command input (only if aircraft selected)
	if g.SelectedAircraft == nil {
		return
	}

	// Command mode selection
	if g.CommandMode == "" {
		// Ground commands (only for holding short / line-up-wait aircraft)
		if g.SelectedAircraft.Phase == aircraft.PhaseHoldingShort {
			if g.InputHandler.IsKeyJustPressed(ebiten.KeyL) {
				atc.IssueLineUpWait(g.SelectedAircraft, g.CommandHistory, g.SimTime)
				return
			}
			if g.InputHandler.IsKeyJustPressed(ebiten.KeyT) {
				atc.IssueTakeoffClearance(g.SelectedAircraft, g.CommandHistory, g.SimTime)
				return
			}
		}
		if g.SelectedAircraft.Phase == aircraft.PhaseLineUpWait {
			if g.InputHandler.IsKeyJustPressed(ebiten.KeyT) {
				atc.IssueTakeoffClearance(g.SelectedAircraft, g.CommandHistory, g.SimTime)
				return
			}
		}

		// Final approach — cleared to land
		if g.SelectedAircraft.Phase == aircraft.PhaseFinal {
			if g.InputHandler.IsKeyJustPressed(ebiten.KeyC) {
				atc.IssueLandingClearance(g.SelectedAircraft, g.CommandHistory, g.SimTime)
				return
			}
		}

		// Airborne commands
		if !g.SelectedAircraft.IsCommandable() {
			return
		}
		if g.InputHandler.IsKeyJustPressed(ebiten.KeyH) {
			g.CommandMode = "HEADING"
			g.CommandInput = ""
		} else if g.InputHandler.IsKeyJustPressed(ebiten.KeyA) {
			g.CommandMode = "ALTITUDE"
			g.CommandInput = ""
		} else if g.InputHandler.IsKeyJustPressed(ebiten.KeyS) {
			g.CommandMode = "SPEED"
			g.CommandInput = ""
		} else if g.InputHandler.IsKeyJustPressed(ebiten.KeyD) {
			g.CommandMode = "DIRECT"
			g.CommandInput = ""
		} else if g.InputHandler.IsKeyJustPressed(ebiten.KeyW) {
			// W = enter holding pattern at current position
			if g.SelectedAircraft.Phase == aircraft.PhaseArrival ||
				g.SelectedAircraft.Phase == aircraft.PhaseDeparture ||
				g.SelectedAircraft.Phase == aircraft.PhaseClimbout {
				g.SelectedAircraft.EnterHold()
			}
		}
		return
	}

	// For DIRECT mode, handle letter input
	if g.CommandMode == "DIRECT" {
		g.handleDirectInput()
		return
	}

	// Number input
	num, pressed := g.InputHandler.GetNumberInput()
	if pressed {
		g.CommandInput += fmt.Sprintf("%d", num)
	}

	// Enter to confirm command
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyEnter) && g.CommandInput != "" {
		g.executeCommand()
		g.CommandMode = ""
		g.CommandInput = ""
	}

	// Backspace to delete last digit
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.CommandInput) > 0 {
		g.CommandInput = g.CommandInput[:len(g.CommandInput)-1]
	}
}

// executeCommand executes the current command
func (g *Game) executeCommand() {
	if g.SelectedAircraft == nil || g.CommandInput == "" {
		return
	}

	var value float64
	fmt.Sscanf(g.CommandInput, "%f", &value)

	switch g.CommandMode {
	case "HEADING":
		if value >= 0 && value < 360 {
			g.SelectedAircraft.DirectTarget = "" // explicit heading clears named target
			atc.IssueHeadingCommand(g.SelectedAircraft, value, g.CommandHistory, g.SimTime)
		}
	case "ALTITUDE":
		value *= 100
		if value >= 0 && value <= g.SelectedAircraft.Type.MaxAltitude {
			atc.IssueAltitudeCommand(g.SelectedAircraft, value, g.CommandHistory, g.SimTime)
		}
	case "SPEED":
		if value >= g.SelectedAircraft.Type.MinSpeed && value <= g.SelectedAircraft.Type.MaxSpeed {
			atc.IssueSpeedCommand(g.SelectedAircraft, value, g.CommandHistory, g.SimTime)
		}
	case "DIRECT":
		// Try STAR/SID route first
		route := airport.FindRoute(g.Routes, g.CommandInput)
		if route != nil {
			g.assignRoute(g.SelectedAircraft, *route)
			g.SelectedAircraft.Commanded = false // let route auto-steer
			if route.Runway != "" {
				rwy, ok := g.Airport.FindRunwayByName(route.Runway)
				if ok {
					threshX, threshY, hdg := airport.GetRunwayThreshold(rwy, route.Runway)
					g.SelectedAircraft.RunwayName = route.Runway
					g.SelectedAircraft.RunwayHeading = hdg
					g.SelectedAircraft.ThresholdX = threshX
					g.SelectedAircraft.ThresholdY = threshY
					g.SelectedAircraft.AirportElevation = g.Airport.Elevation
				}
			}
			break
		}
		// Try waypoint
		wp := airport.FindWaypoint(g.Waypoints, g.CommandInput)
		if wp != nil {
			heading := airport.HeadingTo(g.SelectedAircraft.X, g.SelectedAircraft.Y, wp.X, wp.Y)
			atc.IssueHeadingCommand(g.SelectedAircraft, heading, g.CommandHistory, g.SimTime)
			g.SelectedAircraft.DirectTarget = g.CommandInput
			break
		}
		// Try runway designation (e.g. "13L", "31R")
		rwy, ok := g.Airport.FindRunwayByName(g.CommandInput)
		if ok {
			threshX, threshY, _ := airport.GetRunwayThreshold(rwy, g.CommandInput)
			heading := airport.HeadingTo(g.SelectedAircraft.X, g.SelectedAircraft.Y, threshX, threshY)
			atc.IssueHeadingCommand(g.SelectedAircraft, heading, g.CommandHistory, g.SimTime)
			g.SelectedAircraft.DirectTarget = g.CommandInput
		}
	}
}

// handleDirectInput handles alphanumeric input for DIRECT command
func (g *Game) handleDirectInput() {
	// Letter keys (A-Z)
	letterKeys := map[ebiten.Key]string{
		ebiten.KeyA: "A", ebiten.KeyB: "B", ebiten.KeyC: "C", ebiten.KeyD: "D",
		ebiten.KeyE: "E", ebiten.KeyF: "F", ebiten.KeyG: "G", ebiten.KeyH: "H",
		ebiten.KeyI: "I", ebiten.KeyJ: "J", ebiten.KeyK: "K", ebiten.KeyL: "L",
		ebiten.KeyM: "M", ebiten.KeyN: "N", ebiten.KeyO: "O", ebiten.KeyP: "P",
		ebiten.KeyQ: "Q", ebiten.KeyR: "R", ebiten.KeyS: "S", ebiten.KeyT: "T",
		ebiten.KeyU: "U", ebiten.KeyV: "V", ebiten.KeyW: "W", ebiten.KeyX: "X",
		ebiten.KeyY: "Y", ebiten.KeyZ: "Z",
	}
	for key, letter := range letterKeys {
		if g.InputHandler.IsKeyJustPressed(key) {
			if len(g.CommandInput) < 10 {
				g.CommandInput += letter
			}
			break
		}
	}

	// Digit keys (0-9)
	num, pressed := g.InputHandler.GetNumberInput()
	if pressed && len(g.CommandInput) < 10 {
		g.CommandInput += fmt.Sprintf("%d", num)
	}

	// Enter to confirm
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyEnter) && g.CommandInput != "" {
		g.executeCommand()
		g.CommandMode = ""
		g.CommandInput = ""
	}

	// Backspace to delete last character
	if g.InputHandler.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.CommandInput) > 0 {
		g.CommandInput = g.CommandInput[:len(g.CommandInput)-1]
	}
}

// Draw draws the game
func (g *Game) Draw(screen *ebiten.Image) {
	if g.State == "MENU" {
		g.drawMenu(screen)
		return
	}

	// Sync active runway state to renderer before drawing
	g.Renderer.ActiveLandingRunway = g.ActiveLandingRunway
	g.Renderer.ActiveTakeoffRunway = g.ActiveTakeoffRunway
	g.Renderer.Score = g.Score
	g.Renderer.LandedCount = g.LandedCount
	g.Renderer.Difficulty = g.Difficulty
	g.Renderer.WindDir = g.WindDirection
	g.Renderer.WindSpeed = g.WindSpeed
	g.Renderer.ActiveField = g.ActiveField
	g.Renderer.ActiveAircraft = g.ActiveAircraft
	g.Renderer.MouseX = g.InputHandler.MouseX
	g.Renderer.MouseY = g.InputHandler.MouseY

	// Set radar offset based on panel visibility
	if g.FlightStripPanel != nil && g.FlightStripPanel.Visible {
		g.Renderer.RadarOffsetX = -140 // half of panel width (280/2)
		g.FlightStripPanel.UpdateLayout(g.Renderer.ScreenWidth, g.Renderer.ScreenHeight)
	} else {
		g.Renderer.RadarOffsetX = 0
	}

	g.Renderer.Draw(screen, g.Airport, g.Waypoints, g.Routes, g.Aircraft, g.Conflicts, g.SelectedAircraft, g.SimTime)

	// Draw flight strip panel
	if g.FlightStripPanel != nil && g.FlightStripPanel.Visible {
		g.FlightStripPanel.Draw(screen, g.Aircraft, g.SelectedAircraft,
			g.Renderer.ConflictMap, g.CommandMode, g.CommandInput)
	}

	if g.GameOver {
		sw := float32(g.Renderer.ScreenWidth)
		sh := float32(g.Renderer.ScreenHeight)
		// Full-screen dim overlay
		vector.FillRect(screen, 0, 0, sw, sh, color.RGBA{0, 0, 0, 160}, false)
		// Centered panel
		panelW := float32(340)
		panelH := float32(130)
		panelX := sw/2 - panelW/2
		panelY := sh/2 - panelH/2
		vector.FillRect(screen, panelX, panelY, panelW, panelH, color.RGBA{10, 0, 0, 235}, false)
		vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 2, color.RGBA{220, 0, 0, 255}, false)
		// Title bar
		vector.FillRect(screen, panelX, panelY, panelW, 22, color.RGBA{180, 0, 0, 200}, false)
		textX := int(panelX) + 12
		ebitenutil.DebugPrintAt(screen, "=== GAME OVER ===", textX+50, int(panelY)+4)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Aircraft landed: %d", g.LandedCount), textX, int(panelY)+32)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Final score:     %d", g.Score), textX, int(panelY)+50)
		ebitenutil.DebugPrintAt(screen, "Press R or Enter to return to menu", textX-4, int(panelY)+80)
		return
	}

	// Draw command textbox (hidden when flight strip panel is visible)
	if g.FlightStripPanel == nil || !g.FlightStripPanel.Visible {
		g.CommandTextBox.Draw(screen, g.CommandMode, g.CommandInput, g.SelectedAircraft)
	}

	// Draw runway config menu (on top of everything)
	g.RunwayMenu.Draw(screen, g.ActiveLandingRunway, g.ActiveTakeoffRunway)

	// Draw help overlay (topmost layer)
	if g.ShowHelp {
		g.drawHelp(screen)
	}
}

// drawHelp renders the help overlay on top of the game
func (g *Game) drawHelp(screen *ebiten.Image) {
	sw := float32(g.Renderer.ScreenWidth)
	sh := float32(g.Renderer.ScreenHeight)

	// Dim background
	vector.FillRect(screen, 0, 0, sw, sh, color.RGBA{0, 0, 0, 160}, false)

	// Centered panel
	panelW := float32(540)
	panelH := float32(610)
	px := sw/2 - panelW/2
	py := sh/2 - panelH/2
	vector.FillRect(screen, px, py, panelW, panelH, color.RGBA{0, 12, 24, 245}, false)
	vector.StrokeRect(screen, px, py, panelW, panelH, 2, color.RGBA{0, 200, 255, 220}, false)

	// Title bar
	vector.FillRect(screen, px, py, panelW, 22, color.RGBA{0, 80, 130, 220}, false)
	ebitenutil.DebugPrintAt(screen, "ATC SIMULATOR - HELP", int(px)+panelW2(panelW, "ATC SIMULATOR - HELP"), int(py)+5)

	// Content
	x := int(px) + 20
	y := int(py) + 32
	line := func(text string) {
		ebitenutil.DebugPrintAt(screen, text, x, y)
		y += 16
	}

	line("GOAL")
	line("  Guide arrivals to land and departures out of your")
	line("  sector. Score points for each successful landing")
	line("  (+50) and departure exit (+25).")
	y += 6

	line("SEPARATION")
	line("  Warning:  <5nm horizontal AND <1000ft vertical")
	line("  Critical: <3nm horizontal AND <500ft vertical (-2pts)")
	y += 6

	line("SELECTING AIRCRAFT")
	line("  Left-click an aircraft symbol or its data tag.")
	line("  Click a field on a selected tag, then scroll the")
	line("  mouse wheel to adjust (HDG/ALT/SPD).")
	y += 6

	line("COMMANDS (select aircraft first)")
	line("  H - Heading   (enter 000-359, press Enter)")
	line("  A - Altitude  (enter in 100s, e.g. 50 = 5000ft)")
	line("  S - Speed     (enter knots, press Enter)")
	line("  D - Direct to (waypoint, runway, STAR or SID name)")
	line("  W - Hold      (enter holding pattern)")
	y += 6

	line("LANDING SEQUENCE")
	line("  1. Direct arrival to a STAR or vector toward runway")
	line("  2. Aircraft auto-descends within 15nm when aligned")
	line("  3. ILS captures automatically (within 15nm, <30 off)")
	line("  4. Press C to clear for landing once on ILS")
	y += 6

	line("TAKEOFF SEQUENCE")
	line("  1. Press L to line up and wait")
	line("  2. Press T to clear for takeoff")
	line("  3. Aircraft follows assigned SID automatically")
	y += 6

	line("OTHER CONTROLS")
	line("  Right-click near airport  - runway configuration")
	line("  Right-click label/tag     - reset position")
	line("  Mouse wheel               - zoom in/out")
	line("  Tab                       - toggle flight strip panel")
	line("  ESC                       - deselect / close menus")

	// Footer
	footerY := int(py+panelH) - 20
	vector.StrokeLine(screen, px+10, float32(footerY-4), px+panelW-10, float32(footerY-4), 1, color.RGBA{0, 200, 255, 80}, false)
	ebitenutil.DebugPrintAt(screen, "Press F1 or ESC to close", int(px)+panelW2(panelW, "Press F1 or ESC to close"), footerY)
}

// panelW2 returns the X offset to center text in a panel of given width
func panelW2(panelW float32, text string) int {
	textW := len(text) * 6
	return (int(panelW) - textW) / 2
}

// drawMenu renders the airport selector screen
func (g *Game) drawMenu(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 10, 20, 255})
	sw := float32(g.Renderer.ScreenWidth)
	sh := float32(g.Renderer.ScreenHeight)

	// Centered panel
	panelW := float32(460)
	panelH := float32(240)
	panelX := sw/2 - panelW/2
	panelY := sh/2 - panelH/2
	vector.FillRect(screen, panelX, panelY, panelW, panelH, color.RGBA{0, 12, 24, 245}, false)
	vector.StrokeRect(screen, panelX, panelY, panelW, panelH, 2, color.RGBA{0, 200, 255, 220}, false)

	// Title bar
	titleH := float32(28)
	vector.FillRect(screen, panelX, panelY, panelW, titleH, color.RGBA{0, 80, 130, 220}, false)
	vector.StrokeLine(screen, panelX, panelY+titleH, panelX+panelW, panelY+titleH, 1, color.RGBA{0, 200, 255, 180}, false)
	ebitenutil.DebugPrintAt(screen, "ATC SIMULATOR", int(panelX)+165, int(panelY)+8)

	// Airport list
	type menuEntry struct{ icao, name string }
	var menuAirports []menuEntry
	for _, icao := range g.MenuAirports {
		apt := airport.GetAirport(icao)
		if apt != nil {
			menuAirports = append(menuAirports, menuEntry{apt.ICAO, apt.Name})
		}
	}
	listStartY := int(panelY) + 44
	rowH := 42
	for i, apt := range menuAirports {
		rowY := listStartY + i*rowH
		// Highlight selected row
		if i == g.MenuSelection {
			vector.FillRect(screen, panelX+4, float32(rowY)-2, panelW-8, float32(rowH)-4, color.RGBA{0, 60, 100, 180}, false)
			vector.StrokeRect(screen, panelX+4, float32(rowY)-2, panelW-8, float32(rowH)-4, 1, color.RGBA{0, 200, 255, 150}, false)
			ebitenutil.DebugPrintAt(screen, ">", int(panelX)+8, rowY+14)
		}
		// ICAO code on top line, full airport name below
		ebitenutil.DebugPrintAt(screen, apt.icao, int(panelX)+22, rowY+4)
		ebitenutil.DebugPrintAt(screen, apt.name, int(panelX)+22, rowY+18)
	}

	// Separator
	sepY := float32(listStartY+len(menuAirports)*rowH) + 4
	vector.StrokeLine(screen, panelX+8, sepY, panelX+panelW-8, sepY, 1, color.RGBA{0, 80, 120, 150}, false)

	// Key hints
	ebitenutil.DebugPrintAt(screen, "Click or Up/Down to select  |  Double-click or Enter to start", int(panelX)+8, int(sepY)+8)
}

// Layout updates the renderer to match the actual window size each frame,
// eliminating pixelation when the window is resized.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.Renderer.ScreenWidth = outsideWidth
	g.Renderer.ScreenHeight = outsideHeight
	return outsideWidth, outsideHeight
}
