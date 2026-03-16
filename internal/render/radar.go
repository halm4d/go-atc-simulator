package render

import (
	"atc-sim/internal/aircraft"
	"atc-sim/internal/airport"
	"atc-sim/internal/atc"
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	// Colors matching real ATC displays
	ColorBackground = color.RGBA{18, 26, 42, 255}     // Medium gray-blue (authentic ATC display tone)
	ColorGrid       = color.RGBA{0, 40, 60, 255}     // Subtle grid
	ColorRangeRing  = color.RGBA{0, 80, 120, 255}    // Range rings
	ColorRunway     = color.RGBA{200, 200, 200, 255} // Bright gray runways
	ColorAirport    = color.RGBA{255, 255, 255, 255} // White airport symbol
	ColorWaypoint   = color.RGBA{0, 200, 255, 255}   // Cyan waypoints
	ColorRoute      = color.RGBA{100, 100, 150, 180} // Purple-ish route lines (semi-transparent)
	ColorAircraft       = color.RGBA{0, 220, 80, 255}    // Green — commanded arrival
	ColorUncommanded    = color.RGBA{200, 200, 200, 255}  // White — no commands yet
	ColorDeparture      = color.RGBA{80, 180, 255, 255}  // Sky blue — departure
	ColorGround         = color.RGBA{255, 180, 0, 255}   // Amber — ground/holding
	ColorTakeoffRoll    = color.RGBA{220, 220, 220, 255} // White — rolling for takeoff
	ColorFinal          = color.RGBA{0, 200, 255, 255}   // Cyan — climbout / approach cone
	ColorNeedsLanding   = color.RGBA{255, 130, 0, 255}   // Orange — on ILS, awaiting clearance
	ColorLandingCleared = color.RGBA{180, 255, 180, 255} // Light green — cleared to land
	ColorSelected       = color.RGBA{255, 255, 0, 255}   // Yellow — selected
	ColorDataTag        = color.RGBA{0, 220, 80, 255}    // Green text (default)
	ColorConflict       = color.RGBA{255, 100, 0, 255}   // Orange warning
	ColorCritical       = color.RGBA{255, 0, 0, 255}     // Red critical
	ColorUI             = color.RGBA{200, 200, 200, 255} // UI elements
)

// labelRect tracks a placed label's bounding box for overlap avoidance.
type labelRect struct{ X, Y, W, H int }

// Renderer handles all radar display rendering
type Renderer struct {
	ScreenWidth         int
	ScreenHeight        int
	Scale               float64 // Pixels per nautical mile
	MinScale            float64 // Minimum zoom (zoomed out)
	MaxScale            float64 // Maximum zoom (zoomed in)
	CenterX             float64 // Center X in nm
	CenterY             float64 // Center Y in nm
	RadarOffsetX        int     // Horizontal pixel offset for radar center (negative = shift left)
	ActiveLandingRunway string  // Runway designation used for approach cone
	ActiveTakeoffRunway string

	// HUD data (synced from game state before each Draw)
	Score       int
	LandedCount int
	Difficulty  int
	WindDir     float64
	WindSpeed   float64

	// Per-aircraft conflict severity — populated each frame in Draw(), read by drawDataTag()
	ConflictMap map[*aircraft.Aircraft]string

	// Active data-tag field (synced from game state each frame)
	ActiveField    string
	ActiveAircraft *aircraft.Aircraft

	// Waypoint list — populated each frame in Draw(), used by drawAircraft()
	Waypoints []airport.Waypoint

	// placedLabels accumulates label bounding rects across draw calls within a frame.
	placedLabels []labelRect

	// Draggable labels — waypoints and runway labels
	WaypointLabelOffsets map[string][2]int    // waypoint name → (dx, dy) from marker
	RunwayLabelOffsets   map[string][2]int    // runway name → (dx, dy) from threshold
	LabelRects           map[string]labelRect // all draggable label rects (for hit-testing)
	RouteLabelRects      map[string]labelRect // route name → bounding rect (for drop-target hit-testing)
	DraggingLabel        string               // label key being dragged, "" if none
	DraggingAircraft     *aircraft.Aircraft   // aircraft whose data tag is being dragged
	DragOffsetX          int                  // mouse offset from label origin at drag start
	DragOffsetY          int

	// Drag-to-waypoint state
	DraggingToWP   *aircraft.Aircraft // aircraft being dragged to a waypoint/route
	DragToWPActive bool               // true once drag threshold exceeded
	DropTarget     string             // highlighted drop target ("", "DERUP", "route:STAR1A")
	MouseX, MouseY int                // current mouse position (set before Draw)
}

// NewRenderer creates a new renderer
func NewRenderer(width, height int, scale float64) *Renderer {
	return &Renderer{
		ScreenWidth:          width,
		ScreenHeight:         height,
		Scale:                scale,
		MinScale:             2.0,  // Zoomed out - 2 pixels per nm
		MaxScale:             60.0, // Zoomed in - 60 pixels per nm
		WaypointLabelOffsets: make(map[string][2]int),
		RunwayLabelOffsets:   make(map[string][2]int),
		LabelRects:           make(map[string]labelRect),
		RouteLabelRects:      make(map[string]labelRect),
		CenterX:              0,
		CenterY:      0,
	}
}

// ZoomIn increases the scale (zoom in)
func (r *Renderer) ZoomIn(factor float64) {
	r.Scale *= factor
	if r.Scale > r.MaxScale {
		r.Scale = r.MaxScale
	}
}

// ZoomOut decreases the scale (zoom out)
func (r *Renderer) ZoomOut(factor float64) {
	r.Scale /= factor
	if r.Scale < r.MinScale {
		r.Scale = r.MinScale
	}
}

// GetZoomLevel returns a normalized zoom level (0.0 to 1.0)
func (r *Renderer) GetZoomLevel() float64 {
	return (r.Scale - r.MinScale) / (r.MaxScale - r.MinScale)
}

// GetZoomPercentage returns zoom as a percentage
func (r *Renderer) GetZoomPercentage() int {
	return int(r.GetZoomLevel() * 100)
}

// Draw renders the entire radar display
func (r *Renderer) Draw(screen *ebiten.Image, airportData *airport.Airport, waypoints []airport.Waypoint, routes []airport.Route, aircraftList []*aircraft.Aircraft, conflicts []atc.Conflict, selectedAircraft *aircraft.Aircraft, simTime float64) {
	// Store waypoints for use in drawAircraft()
	r.Waypoints = waypoints

	// Build per-aircraft conflict severity map for CA data-tag annotations
	r.ConflictMap = make(map[*aircraft.Aircraft]string)
	for _, c := range conflicts {
		if existing, ok := r.ConflictMap[c.Aircraft1]; !ok || (existing != "CRITICAL" && c.Severity == "CRITICAL") {
			r.ConflictMap[c.Aircraft1] = c.Severity
		}
		if existing, ok := r.ConflictMap[c.Aircraft2]; !ok || (existing != "CRITICAL" && c.Severity == "CRITICAL") {
			r.ConflictMap[c.Aircraft2] = c.Severity
		}
	}

	// Reset label collision tracking for this frame
	r.placedLabels = r.placedLabels[:0]
	for k := range r.LabelRects {
		delete(r.LabelRects, k)
	}
	for k := range r.RouteLabelRects {
		delete(r.RouteLabelRects, k)
	}

	// Fill background
	screen.Fill(ColorBackground)

	// Draw range rings
	r.drawRangeRings(screen)

	// Draw north indicator
	r.drawNorthIndicator(screen)

	// Draw routes (STAR paths)
	r.drawRoutes(screen, routes, waypoints)

	// Draw airport and runways (before waypoints so runway labels are registered first)
	r.drawAirport(screen, airportData)

	// Draw waypoints (uses r.placedLabels to avoid runway/airport label overlap)
	r.drawWaypoints(screen, waypoints)

	// Draw approach cone for active landing runway
	if r.ActiveLandingRunway != "" {
		r.drawApproachCone(screen, airportData, r.ActiveLandingRunway)
	}

	// Draw conflicts (behind aircraft)
	r.drawConflicts(screen, conflicts, simTime)

	// Draw ILS approach lines (behind aircraft symbols)
	for _, a := range aircraftList {
		if a.Phase == aircraft.PhaseFinal || a.Phase == aircraft.PhaseLanding {
			r.drawApproachLine(screen, a)
		}
	}

	// Draw position trails (behind aircraft symbols)
	for _, a := range aircraftList {
		r.drawTrail(screen, a, a == selectedAircraft)
	}

	// Draw aircraft
	for _, a := range aircraftList {
		r.drawAircraft(screen, a, a == selectedAircraft)
	}

	// Draw drag-to-waypoint visual feedback
	r.DrawDragToWaypoint(screen, waypoints)

	// Draw UI overlay
	r.drawUI(screen, airportData, len(aircraftList), len(conflicts), simTime)
}

// drawRangeRings draws distance rings from center
func (r *Renderer) drawRangeRings(screen *ebiten.Image) {
	centerX := float32(r.radarCenterX())
	centerY := float32(r.ScreenHeight / 2)

	// Draw rings every 10nm
	for i := 1; i <= 6; i++ {
		radius := float32(float64(i) * 10.0 * r.Scale)
		vector.StrokeCircle(screen, centerX, centerY, radius, 1, ColorRangeRing, false)

		// Label at 3 o'clock position (right side of ring) — clean, consistent
		label := fmt.Sprintf("%d", i*10)
		ebitenutil.DebugPrintAt(screen, label, int(centerX+radius)+2, int(centerY)-6)
	}

}

// drawRoutes draws STAR/approach routes
func (r *Renderer) drawRoutes(screen *ebiten.Image, routes []airport.Route, waypoints []airport.Waypoint) {
	// Build a waypoint lookup map
	wpMap := make(map[string]airport.Waypoint)
	for _, wp := range waypoints {
		wpMap[wp.Name] = wp
	}

	// Draw each route
	for _, route := range routes {
		if len(route.Waypoints) < 2 {
			continue
		}

		// Pick color based on route type
		var routeColor, labelColor color.RGBA
		if route.Type == "SID" {
			routeColor = color.RGBA{50, 90, 60, 130}
			labelColor = color.RGBA{70, 130, 80, 200}
		} else {
			routeColor = color.RGBA{50, 60, 110, 130}
			labelColor = color.RGBA{70, 80, 150, 200}
		}

		// Draw lines connecting waypoints
		for i := 0; i < len(route.Waypoints)-1; i++ {
			wp1Name := route.Waypoints[i]
			wp2Name := route.Waypoints[i+1]

			wp1, ok1 := wpMap[wp1Name]
			wp2, ok2 := wpMap[wp2Name]

			if !ok1 || !ok2 {
				continue
			}

			x1, y1 := r.worldToScreen(wp1.X, wp1.Y)
			x2, y2 := r.worldToScreen(wp2.X, wp2.Y)

			vector.StrokeLine(screen, x1, y1, x2, y2, 1, routeColor, false)
		}

		// Draw route name at the entry waypoint (first for STARs, last for SIDs)
		entryIdx := 0
		if route.Type == "SID" {
			entryIdx = len(route.Waypoints) - 1
		}
		entryWP, ok := wpMap[route.Waypoints[entryIdx]]
		if !ok {
			continue
		}
		ex, ey := r.worldToScreen(entryWP.X, entryWP.Y)

		// Offset label along the route direction (away from airport)
		charW := 6
		charH := 12
		lw := len(route.Name) * charW
		lh := charH
		lx, ly := int(ex)-lw/2, int(ey)-lh-8

		// Avoid overlapping already-placed labels
		offsets := [][2]int{
			{-lw / 2, -lh - 8},  // above
			{-lw / 2, 10},       // below
			{10, -lh / 2},       // right
			{-lw - 6, -lh / 2},  // left
		}
		for _, off := range offsets {
			cx, cy := int(ex)+off[0], int(ey)+off[1]
			overlap := false
			for _, p := range r.placedLabels {
				if cx < p.X+p.W && cx+lw > p.X && cy < p.Y+p.H && cy+lh > p.Y {
					overlap = true
					break
				}
			}
			if !overlap {
				lx, ly = cx, cy
				break
			}
		}

		ebitenutil.DebugPrintAt(screen, route.Name, lx, ly)
		routeRect := labelRect{lx, ly, lw, lh}
		r.placedLabels = append(r.placedLabels, routeRect)
		r.RouteLabelRects[route.Name] = routeRect

		// Draw a small leader line from label to entry waypoint
		vector.StrokeLine(screen, float32(lx+lw/2), float32(ly+lh), ex, ey, 1, labelColor, false)
	}
}

// drawWaypoints draws waypoint markers with non-overlapping, draggable labels.
// Uses r.placedLabels to avoid runway and airport labels already registered.
func (r *Renderer) drawWaypoints(screen *ebiten.Image, waypoints []airport.Waypoint) {
	for _, wp := range waypoints {
		screenX, screenY := r.worldToScreen(wp.X, wp.Y)

		// Draw small delta triangle for waypoints (authentic ATC fix symbol)
		size := float32(4)
		wpCol := color.RGBA{0, 160, 200, 160}
		vector.StrokeLine(screen, screenX-size, screenY+size, screenX+size, screenY+size, 1, wpCol, false)
		vector.StrokeLine(screen, screenX+size, screenY+size, screenX, screenY-size, 1, wpCol, false)
		vector.StrokeLine(screen, screenX, screenY-size, screenX-size, screenY+size, 1, wpCol, false)

		charW := 6
		charH := 12
		lw := len(wp.Name) * charW
		lh := charH
		sx, sy := int(screenX), int(screenY)

		var lx, ly int

		if customOff, ok := r.WaypointLabelOffsets[wp.Name]; ok {
			// User has manually positioned this label
			lx, ly = sx+customOff[0], sy+customOff[1]
		} else {
			// Auto-place: pick first non-overlapping candidate
			offsets := [][2]int{
				{8, -lh},              // top-right (default)
				{8, 4},                // bottom-right
				{-lw - 4, -lh},       // top-left
				{-lw - 4, 4},         // bottom-left
				{-lw/2 + 4, -lh - 6}, // centered above
				{-lw/2 + 4, lh},      // centered below
			}

			bestOff := offsets[0]
			for _, off := range offsets {
				rx, ry := sx+off[0], sy+off[1]
				overlaps := false
				for _, p := range r.placedLabels {
					if rx < p.X+p.W && rx+lw > p.X && ry < p.Y+p.H && ry+lh > p.Y {
						overlaps = true
						break
					}
				}
				if !overlaps {
					bestOff = off
					break
				}
			}
			lx, ly = sx+bestOff[0], sy+bestOff[1]
		}

		// Draw leader line from marker to label
		if r.WaypointLabelOffsets[wp.Name] != ([2]int{}) {
			vector.StrokeLine(screen, screenX, screenY, float32(lx), float32(ly+lh/2), 1, color.RGBA{0, 160, 200, 60}, false)
		}

		ebitenutil.DebugPrintAt(screen, wp.Name, lx, ly)
		rect := labelRect{lx, ly, lw, lh}
		r.placedLabels = append(r.placedLabels, rect)
		r.LabelRects[wp.Name] = rect
	}
}

// GetLabelAt returns the key of the label at (mx, my), or "".
// Keys are waypoint names (e.g. "DERUP") or "rwy:13R" for runway labels.
func (r *Renderer) GetLabelAt(mx, my int) string {
	for key, rect := range r.LabelRects {
		if mx >= rect.X && mx <= rect.X+rect.W && my >= rect.Y && my <= rect.Y+rect.H {
			return key
		}
	}
	return ""
}

// GetRouteLabelAt returns the route name whose label contains (mx, my), or "".
func (r *Renderer) GetRouteLabelAt(mx, my int) string {
	for name, rect := range r.RouteLabelRects {
		if mx >= rect.X && mx <= rect.X+rect.W && my >= rect.Y && my <= rect.Y+rect.H {
			return name
		}
	}
	return ""
}

// GetWaypointNear returns the waypoint name nearest to (mx, my) within radiusPx, or "".
func (r *Renderer) GetWaypointNear(mx, my int, radiusPx float64, waypoints []airport.Waypoint) string {
	bestDist := radiusPx
	bestName := ""
	for _, wp := range waypoints {
		sx, sy := r.worldToScreen(wp.X, wp.Y)
		dx := float64(mx) - float64(sx)
		dy := float64(my) - float64(sy)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < bestDist {
			bestDist = dist
			bestName = wp.Name
		}
	}
	return bestName
}

// DrawDragToWaypoint renders the drag-to-waypoint visual feedback.
func (r *Renderer) DrawDragToWaypoint(screen *ebiten.Image, waypoints []airport.Waypoint) {
	if !r.DragToWPActive || r.DraggingToWP == nil {
		return
	}
	a := r.DraggingToWP
	ax, ay := r.worldToScreen(a.X, a.Y)

	// Draw line from aircraft to cursor
	lineCol := color.RGBA{255, 255, 0, 180}
	vector.StrokeLine(screen, ax, ay, float32(r.MouseX), float32(r.MouseY), 2, lineCol, false)

	// Highlight drop target
	if r.DropTarget != "" {
		highlightCol := color.RGBA{0, 255, 200, 200}
		if len(r.DropTarget) > 6 && r.DropTarget[:6] == "route:" {
			routeName := r.DropTarget[6:]
			if rect, ok := r.RouteLabelRects[routeName]; ok {
				vector.StrokeRect(screen, float32(rect.X-2), float32(rect.Y-2),
					float32(rect.W+4), float32(rect.H+4), 2, highlightCol, false)
			}
		} else {
			// Waypoint highlight: ring around the waypoint marker
			for _, wp := range waypoints {
				if wp.Name == r.DropTarget {
					sx, sy := r.worldToScreen(wp.X, wp.Y)
					vector.StrokeCircle(screen, sx, sy, 12, 2, highlightCol, false)
					// Draw name prominently near the ring
					ebitenutil.DebugPrintAt(screen, wp.Name, int(sx)+14, int(sy)-6)
					break
				}
			}
		}
	}
}

// drawAirport draws the airport and runways
func (r *Renderer) drawAirport(screen *ebiten.Image, airport *airport.Airport) {
	centerX := float32(r.radarCenterX())
	centerY := float32(r.ScreenHeight / 2)

	// Draw runways first (behind airport symbol)
	for _, runway := range airport.Runways {
		r.drawRunway(screen, centerX, centerY, runway)
	}

	// Draw airport symbol — small cross + circle, unobtrusive
	vector.StrokeLine(screen, centerX-8, centerY, centerX+8, centerY, 1, color.RGBA{180, 180, 180, 200}, false)
	vector.StrokeLine(screen, centerX, centerY-8, centerX, centerY+8, 1, color.RGBA{180, 180, 180, 200}, false)
	vector.StrokeCircle(screen, centerX, centerY, 10, 2, color.RGBA{180, 180, 180, 200}, false)

	// Draw airport label - larger and more visible
	label := fmt.Sprintf("  %s  ", airport.ICAO)
	// Draw background for label
	labelX := int(centerX) + 30
	labelY := int(centerY) - 25
	vector.DrawFilledRect(screen, float32(labelX-2), float32(labelY-2), 50, 14, ColorBackground, false)
	ebitenutil.DebugPrintAt(screen, label, labelX, labelY)
	r.placedLabels = append(r.placedLabels, labelRect{labelX - 2, labelY - 2, 54, 18})
}

// drawRunway draws a single runway
func (r *Renderer) drawRunway(screen *ebiten.Image, airportX, airportY float32, runway airport.Runway) {
	// Draw runway as a thick line from center
	lengthNm := runway.Length / 6076.0 // Convert feet to nm
	lengthPx := float32(lengthNm * r.Scale)

	// Convert heading to radians (0° = North, clockwise)
	angle1 := float32((90 - runway.Heading1) * math.Pi / 180)
	angle2 := float32((90 - runway.Heading2) * math.Pi / 180)

	// Calculate endpoints
	x1 := airportX + lengthPx/2*float32(math.Cos(float64(angle1)))
	y1 := airportY - lengthPx/2*float32(math.Sin(float64(angle1)))
	x2 := airportX + lengthPx/2*float32(math.Cos(float64(angle2)))
	y2 := airportY - lengthPx/2*float32(math.Sin(float64(angle2)))

	// Draw runway with width representation
	widthFt := runway.Width
	widthPx := float32(widthFt / 6076.0 * r.Scale * 3) // Scale width visually (3x multiplier for visibility)
	if widthPx < 8 {
		widthPx = 8 // Minimum width for visibility
	}

	// Draw main runway body (thick white line)
	vector.StrokeLine(screen, x1, y1, x2, y2, widthPx, ColorRunway, false)

	// Draw center line (dashed effect with darker color)
	dashColor := color.RGBA{100, 100, 100, 255}
	vector.StrokeLine(screen, x1, y1, x2, y2, 2, dashColor, false)

	// Draw runway edge lines for definition
	perpAngle := angle1 + float32(math.Pi/2)
	edgeOffset := widthPx / 2

	// Edge 1
	edge1x1 := x1 + edgeOffset*float32(math.Cos(float64(perpAngle)))
	edge1y1 := y1 - edgeOffset*float32(math.Sin(float64(perpAngle)))
	edge1x2 := x2 + edgeOffset*float32(math.Cos(float64(perpAngle)))
	edge1y2 := y2 - edgeOffset*float32(math.Sin(float64(perpAngle)))

	// Edge 2
	edge2x1 := x1 - edgeOffset*float32(math.Cos(float64(perpAngle)))
	edge2y1 := y1 + edgeOffset*float32(math.Sin(float64(perpAngle)))
	edge2x2 := x2 - edgeOffset*float32(math.Cos(float64(perpAngle)))
	edge2y2 := y2 + edgeOffset*float32(math.Sin(float64(perpAngle)))

	edgeColor := color.RGBA{255, 255, 255, 180}
	vector.StrokeLine(screen, edge1x1, edge1y1, edge1x2, edge1y2, 1, edgeColor, false)
	vector.StrokeLine(screen, edge2x1, edge2y1, edge2x2, edge2y2, 1, edgeColor, false)

	// Draw runway number boxes with backgrounds
	r.drawRunwayLabel(screen, runway.Name1, int(x1), int(y1), ColorRunway)
	r.drawRunwayLabel(screen, runway.Name2, int(x2), int(y2), ColorRunway)
}

// drawApproachCone draws the 30° ILS capture cone for the given runway.
// Two boundary lines at ±15° and a solid center line, all extending 15 nm from threshold.
func (r *Renderer) drawApproachCone(screen *ebiten.Image, apt *airport.Airport, runwayName string) {
	rwy, ok := apt.FindRunwayByName(runwayName)
	if !ok {
		return
	}

	threshX, threshY, landingHdg := airport.GetRunwayThreshold(rwy, runwayName)
	tx, ty := r.worldToScreen(threshX, threshY)

	// Aircraft approach from the opposite direction of the landing heading
	approachFrom := landingHdg + 180
	if approachFrom >= 360 {
		approachFrom -= 360
	}

	coneNm := 15.0 // length of cone lines in nm

	centerCol := color.RGBA{170, 80, 255, 200}   // solid purple — localizer course
	boundaryCol := color.RGBA{170, 80, 255, 100} // dim — cone edges

	for i, offset := range []float64{-15, 0, 15} {
		hdg := approachFrom + offset
		rad := (90 - hdg) * math.Pi / 180
		endX := threshX + coneNm*math.Cos(rad)
		endY := threshY + coneNm*math.Sin(rad)
		ex, ey := r.worldToScreen(endX, endY)

		col := boundaryCol
		width := float32(1)
		if i == 1 { // center line
			col = centerCol
			width = 2
		}
		vector.StrokeLine(screen, tx, ty, ex, ey, width, col, false)
	}

	// Arc connecting the two boundary tips (straight line approximation)
	rad0 := (90 - (approachFrom - 15)) * math.Pi / 180
	rad2 := (90 - (approachFrom + 15)) * math.Pi / 180
	l0x, l0y := r.worldToScreen(threshX+coneNm*math.Cos(rad0), threshY+coneNm*math.Sin(rad0))
	l2x, l2y := r.worldToScreen(threshX+coneNm*math.Cos(rad2), threshY+coneNm*math.Sin(rad2))
	vector.StrokeLine(screen, l0x, l0y, l2x, l2y, 1, boundaryCol, false)

	// Label: runway name at the tip of the center line
	rad1 := (90 - approachFrom) * math.Pi / 180
	tipX := threshX + coneNm*math.Cos(rad1)
	tipY := threshY + coneNm*math.Sin(rad1)
	lx, ly := r.worldToScreen(tipX, tipY)
	ebitenutil.DebugPrintAt(screen, "ILS "+runwayName, int(lx)-12, int(ly)-8)
}

// drawRunwayLabel draws a runway number with background box (draggable)
func (r *Renderer) drawRunwayLabel(screen *ebiten.Image, label string, x, y int, col color.RGBA) {
	boxWidth := float32(len(label) * 8)
	boxHeight := float32(12)

	// Default offset: centered above the threshold
	defaultOffX := int(-boxWidth / 2)
	defaultOffY := -20
	offX, offY := defaultOffX, defaultOffY
	if custom, ok := r.RunwayLabelOffsets[label]; ok {
		offX, offY = custom[0], custom[1]
	}

	boxX := float32(x + offX)
	boxY := float32(y + offY)

	// Leader line if manually repositioned
	if _, ok := r.RunwayLabelOffsets[label]; ok {
		vector.StrokeLine(screen, float32(x), float32(y), boxX+boxWidth/2, boxY+boxHeight/2, 1, color.RGBA{180, 180, 180, 60}, false)
	}

	// Dark background with border
	vector.DrawFilledRect(screen, boxX-2, boxY-2, boxWidth+4, boxHeight+4, ColorBackground, false)
	vector.StrokeRect(screen, boxX-2, boxY-2, boxWidth+4, boxHeight+4, 1, col, false)

	// Draw text
	ebitenutil.DebugPrintAt(screen, label, int(boxX), int(boxY))

	// Register for overlap avoidance and hit-testing
	key := "rwy:" + label
	rect := labelRect{int(boxX) - 2, int(boxY) - 2, int(boxWidth) + 4, int(boxHeight) + 4}
	r.placedLabels = append(r.placedLabels, rect)
	r.LabelRects[key] = rect
}

// drawApproachLine draws a dashed line from the aircraft to the runway threshold
func (r *Renderer) drawApproachLine(screen *ebiten.Image, a *aircraft.Aircraft) {
	ax, ay := r.worldToScreen(a.X, a.Y)
	tx, ty := r.worldToScreen(a.ThresholdX, a.ThresholdY)

	lineColor := ColorFinal
	if a.Phase == aircraft.PhaseLanding {
		lineColor = color.RGBA{0, 255, 100, 200} // bright green when cleared
	}
	lineColor.A = 160

	// Draw as a simple dashed effect: 3 segments
	for i := 0; i < 3; i++ {
		t0 := float32(i) / 3.0
		t1 := float32(i)/3.0 + 0.15
		x0 := ax + (tx-ax)*t0
		y0 := ay + (ty-ay)*t0
		x1 := ax + (tx-ax)*t1
		y1 := ay + (ty-ay)*t1
		vector.StrokeLine(screen, x0, y0, x1, y1, 1, lineColor, false)
	}
}

// AircraftColor returns the appropriate color for an aircraft based on its phase and state.
func AircraftColor(a *aircraft.Aircraft, isSelected bool) color.RGBA {
	if isSelected {
		return ColorSelected
	}
	switch a.Phase {
	case aircraft.PhaseHoldingShort, aircraft.PhaseLineUpWait:
		return ColorGround
	case aircraft.PhaseTakeoffRoll:
		return ColorTakeoffRoll
	case aircraft.PhaseHolding:
		return color.RGBA{255, 200, 0, 255} // Amber-yellow for holding pattern
	case aircraft.PhaseFinal:
		return ColorNeedsLanding
	case aircraft.PhaseLanding:
		return ColorLandingCleared
	default:
		if a.IsDeparture {
			return ColorDeparture // Sky blue for all departure phases
		}
		if a.Commanded {
			return ColorAircraft // Green for commanded arrivals
		}
		return ColorUncommanded // White for uncommanded arrivals
	}
}

// drawAircraft draws an aircraft symbol and data tag
func (r *Renderer) drawAircraft(screen *ebiten.Image, a *aircraft.Aircraft, isSelected bool) {
	// Ground phases: square symbol, amber color
	if a.Phase == aircraft.PhaseHoldingShort || a.Phase == aircraft.PhaseLineUpWait {
		r.drawGroundAircraft(screen, a, isSelected)
		return
	}

	screenX, screenY := r.worldToScreen(a.X, a.Y)

	col := AircraftColor(a, isSelected)

	// Triangle aircraft symbol — tip points in heading direction (real ATC secondary target)
	angle := (90 - a.Heading) * math.Pi / 180
	cosA := float32(math.Cos(angle))
	sinA := float32(math.Sin(angle))

	tipLen := float64(9)
	wingLen := float64(6)
	if isSelected {
		tipLen = 11
		wingLen = 8
	}
	wingSpread := 140.0 * math.Pi / 180.0

	// Three vertices: forward tip, left wing, right wing
	tipX := screenX + float32(tipLen)*cosA
	tipY := screenY - float32(tipLen)*sinA
	lwX := screenX + float32(wingLen)*float32(math.Cos(angle+wingSpread))
	lwY := screenY - float32(wingLen)*float32(math.Sin(angle+wingSpread))
	rwX := screenX + float32(wingLen)*float32(math.Cos(angle-wingSpread))
	rwY := screenY - float32(wingLen)*float32(math.Sin(angle-wingSpread))

	vector.StrokeLine(screen, tipX, tipY, lwX, lwY, 1.5, col, false)
	vector.StrokeLine(screen, lwX, lwY, rwX, rwY, 1.5, col, false)
	vector.StrokeLine(screen, rwX, rwY, tipX, tipY, 1.5, col, false)

	// Selection ring instead of larger symbol
	if isSelected {
		vector.StrokeCircle(screen, screenX, screenY, 14, 2, ColorSelected, false)
	}

	// Velocity vector: 1-minute look-ahead, proportional to current speed
	lookAheadNm := a.Speed / 60.0
	vectorPx := float32(lookAheadNm * r.Scale)
	if vectorPx > 80 {
		vectorPx = 80
	}
	velCol := col
	velCol.A = uint8(float32(col.A) * 0.6)
	vector.StrokeLine(screen, tipX, tipY, tipX+vectorPx*cosA, tipY-vectorPx*sinA, 1, velCol, false)

	// Target heading dashed vector — only when actually turning (diff > 2°)
	hdgDiff := math.Abs(a.TargetHeading - a.Heading)
	if hdgDiff > 180 {
		hdgDiff = 360 - hdgDiff
	}
	if hdgDiff > 2 {
		tgtAngle := (90 - a.TargetHeading) * math.Pi / 180
		tgtCos := float32(math.Cos(tgtAngle))
		tgtSin := float32(math.Sin(tgtAngle))
		tgtEndX := tipX + vectorPx*tgtCos
		tgtEndY := tipY - vectorPx*tgtSin
		tgtCol := color.RGBA{180, 220, 255, 140} // dim light-blue
		for i := 0; i < 2; i++ {
			t0 := float32(i) / 2.0
			t1 := t0 + 0.35
			sx0 := tipX + (tgtEndX-tipX)*t0
			sy0 := tipY + (tgtEndY-tipY)*t0
			sx1 := tipX + (tgtEndX-tipX)*t1
			sy1 := tipY + (tgtEndY-tipY)*t1
			vector.StrokeLine(screen, sx0, sy0, sx1, sy1, 1, tgtCol, false)
		}
	}

	// Waypoint connection line — aircraft to its route waypoints or direct-to target
	// Skip for ground and approach/landing phases (approach cone already covers those)
	skipPhases := a.Phase == aircraft.PhaseHoldingShort ||
		a.Phase == aircraft.PhaseLineUpWait ||
		a.Phase == aircraft.PhaseTakeoffRoll ||
		a.Phase == aircraft.PhaseFinal ||
		a.Phase == aircraft.PhaseLanding
	if !skipPhases {
		lineCol := color.RGBA{0, 160, 200, 90} // dim cyan
		if a.HasRoute && len(a.RouteWaypoints) > 0 {
			// Draw polyline from aircraft through all remaining route waypoints
			prevX, prevY := screenX, screenY
			for _, wp := range a.RouteWaypoints {
				wx, wy := r.worldToScreen(wp[0], wp[1])
				r.drawDashedLine(screen, prevX, prevY, wx, wy, lineCol)
				prevX, prevY = wx, wy
			}
		} else if a.DirectTarget != "" {
			for i := range r.Waypoints {
				if r.Waypoints[i].Name == a.DirectTarget {
					tx, ty := r.worldToScreen(r.Waypoints[i].X, r.Waypoints[i].Y)
					r.drawDashedLine(screen, screenX, screenY, tx, ty, lineCol)
					break
				}
			}
		}
	}

	r.drawDataTag(screen, a, screenX, screenY, isSelected)
}

// drawDashedLine draws a dashed line between two screen points.
func (r *Renderer) drawDashedLine(screen *ebiten.Image, x0, y0, x1, y1 float32, col color.RGBA) {
	for i := 0; i < 3; i++ {
		t0 := float32(i) / 3.0
		t1 := t0 + 0.22
		sx := x0 + (x1-x0)*t0
		sy := y0 + (y1-y0)*t0
		ex := x0 + (x1-x0)*t1
		ey := y0 + (y1-y0)*t1
		vector.StrokeLine(screen, sx, sy, ex, ey, 1, col, false)
	}
}

// drawTrail draws the historical position trail as discrete fading dots (authentic ATC style)
func (r *Renderer) drawTrail(screen *ebiten.Image, a *aircraft.Aircraft, isSelected bool) {
	n := len(a.Trail)
	if n < 1 {
		return
	}
	for i, pos := range a.Trail {
		// Older dots are more transparent; newest dot at alpha ~160
		alpha := uint8(30 + 130*i/n)
		dotCol := color.RGBA{40, 80, 200, alpha}
		sx, sy := r.worldToScreen(pos[0], pos[1])
		vector.DrawFilledCircle(screen, sx, sy, 2, dotCol, false)
	}
}

// drawGroundAircraft draws a square symbol for ground / holding-short aircraft
func (r *Renderer) drawGroundAircraft(screen *ebiten.Image, a *aircraft.Aircraft, isSelected bool) {
	screenX, screenY := r.worldToScreen(a.X, a.Y)

	col := AircraftColor(a, isSelected)

	size := float32(6)
	if isSelected {
		size = 9
	}
	vector.StrokeRect(screen, screenX-size/2, screenY-size/2, size, size, 2, col, false)

	r.drawDataTag(screen, a, screenX, screenY, isSelected)
}

// altTrend returns +1 (climbing), -1 (descending), 0 (level).
func altTrend(a *aircraft.Aircraft) int {
	diff := a.TargetAltitude - a.Altitude
	if math.Abs(diff) < 100 {
		return 0
	}
	if diff > 0 {
		return 1
	}
	return -1
}

// spdTrend returns +1 (speeding up), -1 (slowing down), 0 (stable).
func spdTrend(a *aircraft.Aircraft) int {
	diff := a.TargetSpeed - a.Speed
	if math.Abs(diff) < 5 {
		return 0
	}
	if diff > 0 {
		return 1
	}
	return -1
}

// drawTrendArrow draws a recognisable up/down arrow (stem + arrowhead) at
// pixel (x, y). trend +1 → bright-green ↑, trend -1 → red ↓, 0 → nothing.
// The arrow fits in a ~7 × 13 px cell (same size as one debug-font character).
func drawTrendArrow(screen *ebiten.Image, x, y int, trend int) {
	if trend == 0 {
		return
	}
	mx := float32(x) + 3.5 // horizontal centre
	top := float32(y) + 1
	bot := float32(y) + 12

	if trend > 0 {
		col := color.RGBA{0, 255, 80, 255} // bright green
		// Stem: bottom → near top
		vector.StrokeLine(screen, mx, bot, mx, top+4, 2, col, false)
		// Arrowhead: two diagonals meeting at the tip
		vector.StrokeLine(screen, mx-4, top+5, mx, top, 2, col, false)
		vector.StrokeLine(screen, mx+4, top+5, mx, top, 2, col, false)
	} else {
		col := color.RGBA{255, 60, 60, 255} // red
		// Stem: top → near bottom
		vector.StrokeLine(screen, mx, top, mx, bot-4, 2, col, false)
		// Arrowhead: two diagonals meeting at the tip
		vector.StrokeLine(screen, mx-4, bot-5, mx, bot, 2, col, false)
		vector.StrokeLine(screen, mx+4, bot-5, mx, bot, 2, col, false)
	}
}

// drawDataTag draws the aircraft data tag with per-row rendering so trend
// arrows can be drawn in green (up) or red (down) between the current and
// target values.
func (r *Renderer) drawDataTag(screen *ebiten.Image, a *aircraft.Aircraft, x, y float32, isSelected bool) {
	col := AircraftColor(a, isSelected)

	// Calculate tag position from free-form offset
	tagX := int(x) + a.DataTagOffX
	tagY := int(y) + a.DataTagOffY

	// Draw tag line (leader line from aircraft to tag)
	vector.StrokeLine(screen, x, y, float32(tagX), float32(tagY+5), 1, col, false)

	// Dark backing rectangle for readability
	// Width 136: covers row-2 "ALT→ALT SPD→SPD PHASE" (~125px max) + 6px padding
	// Height 54: 3 rows × 16px line-height + 6px padding
	vector.DrawFilledRect(screen, float32(tagX-3), float32(tagY-3), 136, 54, color.RGBA{0, 5, 15, 180}, false)
	vector.StrokeRect(screen, float32(tagX-3), float32(tagY-3), 136, 54, 1, color.RGBA{0, 60, 90, 120}, false)

	// Active-field highlight row
	if r.ActiveAircraft == a && r.ActiveField != "" {
		highlightY := float32(tagY + 13) // row 2 (ALT/SPD)
		if r.ActiveField == "HEADING" {
			highlightY = float32(tagY + 29) // row 3
		}
		vector.DrawFilledRect(screen, float32(tagX-3), highlightY, 136, 16, color.RGBA{0, 120, 180, 80}, false)
		vector.StrokeRect(screen, float32(tagX-3), highlightY, 136, 16, 1, color.RGBA{0, 200, 255, 160}, false)
	}

	// CA conflict alert badge above the tag
	if severity, ok := r.ConflictMap[a]; ok {
		caColor := color.RGBA{255, 100, 0, 230}
		if severity == "CRITICAL" {
			caColor = color.RGBA{255, 30, 30, 240}
		}
		vector.DrawFilledRect(screen, float32(tagX-3), float32(tagY-16), 18, 12, caColor, false)
		ebitenutil.DebugPrintAt(screen, "CA", tagX-2, tagY-15)
	}

	row1 := fmt.Sprintf("%s %s", a.Callsign, a.Type.ICAO)

	// Row 1 and row 3 are plain text drawn at fixed offsets.
	// Row 2 is drawn piece-by-piece so colored arrows can be injected.
	const lineH = 16 // Ebiten debug font line height
	const cw = 6    // approximate character advance in pixels

	ebitenutil.DebugPrintAt(screen, row1, tagX, tagY)

	row2Y := tagY + lineH

	switch a.Phase {
	case aircraft.PhaseHoldingShort:
		ebitenutil.DebugPrintAt(screen, a.RunwayName+" HOLD", tagX, row2Y)
	case aircraft.PhaseLineUpWait:
		ebitenutil.DebugPrintAt(screen, a.RunwayName+" L+W", tagX, row2Y)
	case aircraft.PhaseTakeoffRoll:
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("T/O %dkt", int(a.Speed)), tagX, row2Y)
	default:
		// Altitude segment: "CUR [arrow] [TGT]"
		altBase := fmt.Sprintf("%03d", int(a.Altitude/100))
		at := altTrend(a)
		curX := tagX
		ebitenutil.DebugPrintAt(screen, altBase, curX, row2Y)
		curX += len(altBase) * cw
		const arrowW = 8 // px reserved for each trend arrow
		if at != 0 {
			drawTrendArrow(screen, curX, row2Y, at)
			curX += arrowW
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%03d", int(a.TargetAltitude/100)), curX, row2Y)
			curX += 3 * cw
		}
		curX += cw // space between alt and spd

		// Speed segment in tens of knots (real STARS format: "28" = 280 kt)
		spdBase := fmt.Sprintf("%02d", int(a.Speed)/10)
		st := spdTrend(a)
		ebitenutil.DebugPrintAt(screen, spdBase, curX, row2Y)
		curX += len(spdBase) * cw
		if st != 0 {
			drawTrendArrow(screen, curX, row2Y, st)
			curX += arrowW
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%02d", int(a.TargetSpeed)/10), curX, row2Y)
			curX += 2 * cw
		}

		// Phase suffix
		switch a.Phase {
		case aircraft.PhaseHolding:
			ebitenutil.DebugPrintAt(screen, " HOLD", curX, row2Y)
		case aircraft.PhaseClimbout:
			ebitenutil.DebugPrintAt(screen, " CLB", curX, row2Y)
		case aircraft.PhaseFinal:
			ebitenutil.DebugPrintAt(screen, " ILS", curX, row2Y)
		case aircraft.PhaseLanding:
			ebitenutil.DebugPrintAt(screen, " LND", curX, row2Y)
		}
	}

	// Row 3: direct target / SID / heading, with target heading shown when turning
	row3Y := tagY + 2*lineH
	if a.HasRoute && a.RouteName != "" {
		label := a.RouteName
		if len(a.RouteNames) > 0 {
			label += " > " + a.RouteNames[0]
		}
		ebitenutil.DebugPrintAt(screen, label, tagX, row3Y)
	} else if a.DirectTarget != "" {
		ebitenutil.DebugPrintAt(screen, "> "+a.DirectTarget, tagX, row3Y)
	} else {
		// "HDG CUR" — append ">TGT" when turning (diff > 2°)
		curX3 := tagX
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("HDG %03d", int(a.Heading)), curX3, row3Y)
		curX3 += 7 * cw // "HDG 000" = 7 chars
		hdgDiff := math.Abs(a.TargetHeading - a.Heading)
		if hdgDiff > 180 {
			hdgDiff = 360 - hdgDiff
		}
		if hdgDiff > 2 {
			ebitenutil.DebugPrintAt(screen, ">", curX3, row3Y)
			curX3 += cw
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%03d", int(a.TargetHeading)), curX3, row3Y)
		}
	}
}

// GetDataTagBounds returns the bounding box of a data tag for click detection
func (r *Renderer) GetDataTagBounds(a *aircraft.Aircraft, x, y float32) (int, int, int, int) {
	tagX := int(x) + a.DataTagOffX
	tagY := int(y) + a.DataTagOffY
	// 3 rows × ~13 px each ≈ 38 px tall, 110 px wide (wider for alt/spd targets)
	return tagX, tagY, tagX + 110, tagY + 38
}

// drawConflicts draws a pulsing conflict circle on each aircraft involved in a conflict
func (r *Renderer) drawConflicts(screen *ebiten.Image, conflicts []atc.Conflict, simTime float64) {
	// Track worst severity per aircraft (true = CRITICAL)
	isCritical := make(map[*aircraft.Aircraft]bool)
	for _, c := range conflicts {
		if c.Severity == "CRITICAL" {
			isCritical[c.Aircraft1] = true
			isCritical[c.Aircraft2] = true
		} else {
			if _, ok := isCritical[c.Aircraft1]; !ok {
				isCritical[c.Aircraft1] = false
			}
			if _, ok := isCritical[c.Aircraft2]; !ok {
				isCritical[c.Aircraft2] = false
			}
		}
	}

	pulse := float32(math.Abs(math.Sin(simTime * 3)))
	baseRadius := float32(16)

	for a, crit := range isCritical {
		sx, sy := r.worldToScreen(a.X, a.Y)
		radius := baseRadius + pulse*5

		var fillColor, ringColor color.RGBA
		if crit {
			fillColor = color.RGBA{255, 50, 50, uint8(60 + int(pulse*80))}
			ringColor = color.RGBA{255, 0, 0, uint8(180 + int(pulse*75))}
		} else {
			fillColor = color.RGBA{255, 120, 0, uint8(50 + int(pulse*60))}
			ringColor = color.RGBA{255, 100, 0, 200}
		}

		vector.DrawFilledCircle(screen, sx, sy, radius, fillColor, false)
		vector.StrokeCircle(screen, sx, sy, radius, 2, ringColor, false)
	}
}

// drawUI draws the UI overlay with panel bars
func (r *Renderer) drawUI(screen *ebiten.Image, airport *airport.Airport, aircraftCount, conflictCount int, simTime float64) {
	sw := float32(r.ScreenWidth)
	sh := float32(r.ScreenHeight)
	panelBg := color.RGBA{0, 5, 15, 210}
	panelBorder := color.RGBA{0, 80, 120, 200}

	// Top panel bar
	vector.DrawFilledRect(screen, 0, 0, sw, 48, panelBg, false)
	vector.StrokeLine(screen, 0, 48, sw, 48, 1, panelBorder, false)

	// Bottom panel bar
	vector.DrawFilledRect(screen, 0, sh-22, sw, 22, panelBg, false)
	vector.StrokeLine(screen, 0, sh-22, sw, sh-22, 1, panelBorder, false)

	// Wind rose (top-left, inside panel)
	r.drawWindRose(screen, 22, 24, r.WindDir, r.WindSpeed)

	// Status: airport, aircraft count, conflicts, time
	// Start at x=76 to clear the wind rose (radius 13) + speed label (~24px wide) ending at ~x=63
	header := fmt.Sprintf("%s | AC: %d | SEP: %d | T: %.0fs",
		airport.Name, aircraftCount, conflictCount, simTime)
	ebitenutil.DebugPrintAt(screen, header, 76, 8)

	// Zoom level
	zoomText := fmt.Sprintf("Zoom: %.1f px/nm (%d%%)", r.Scale, r.GetZoomPercentage())
	ebitenutil.DebugPrintAt(screen, zoomText, 76, 26)

	// Score / landed / difficulty (top-right, offset by panel if visible)
	scoreText := fmt.Sprintf("Score: %d  Landed: %d  Diff: %d", r.Score, r.LandedCount, r.Difficulty)
	scoreRight := r.ScreenWidth
	if r.RadarOffsetX < 0 {
		scoreRight = r.ScreenWidth + r.RadarOffsetX*2 // stay left of panel
	}
	scoreX := scoreRight - len(scoreText)*6 - 10
	ebitenutil.DebugPrintAt(screen, scoreText, scoreX, 18)

	// Instructions at bottom
	instructions := "Ground: L=LineUp  T=Takeoff | Air: H=Hdg  A=Alt  S=Spd  D=Direct  W=Hold | Final: C=Land | Wheel=Zoom | Tab=Panel | ESC=Deselect"
	ebitenutil.DebugPrintAt(screen, instructions, 10, r.ScreenHeight-16)
}

// drawWindRose draws a small wind direction compass at the given center position
func (r *Renderer) drawWindRose(screen *ebiten.Image, cx, cy int, windDir, windSpeed float64) {
	radius := float32(13)
	cx32, cy32 := float32(cx), float32(cy)

	// Outer ring
	vector.StrokeCircle(screen, cx32, cy32, radius, 1, color.RGBA{0, 160, 200, 180}, false)

	// Arrow pointing toward where wind comes FROM
	// windDir=0 (FROM North) → tip points up (negative screen Y)
	rad := (90 - windDir) * math.Pi / 180
	arrowLen := radius - 2
	tipX := cx32 + float32(math.Cos(rad))*arrowLen
	tipY := cy32 - float32(math.Sin(rad))*arrowLen
	tailX := cx32 - float32(math.Cos(rad))*(arrowLen*0.5)
	tailY := cy32 + float32(math.Sin(rad))*(arrowLen*0.5)

	arrowCol := color.RGBA{0, 220, 255, 230}
	vector.StrokeLine(screen, tailX, tailY, tipX, tipY, 2, arrowCol, false)

	// Arrowhead (two short lines from the tip)
	perpRad := rad + math.Pi/2
	ahLen := float32(4)
	vector.StrokeLine(screen, tipX, tipY,
		tipX-ahLen*float32(math.Cos(rad))+ahLen*0.5*float32(math.Cos(perpRad)),
		tipY+ahLen*float32(math.Sin(rad))-ahLen*0.5*float32(math.Sin(perpRad)),
		2, arrowCol, false)
	vector.StrokeLine(screen, tipX, tipY,
		tipX-ahLen*float32(math.Cos(rad))-ahLen*0.5*float32(math.Cos(perpRad)),
		tipY+ahLen*float32(math.Sin(rad))+ahLen*0.5*float32(math.Sin(perpRad)),
		2, arrowCol, false)

	// Wind speed label to the right of the rose
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%02.0fkt", windSpeed), cx+17, cy-6)
}

// drawNorthIndicator draws a small N↑ marker at the top-center of the radar display
func (r *Renderer) drawNorthIndicator(screen *ebiten.Image) {
	cx := float32(r.radarCenterX())
	// Place just below the top HUD bar
	baseY := float32(62)
	tickLen := float32(14)
	col := color.RGBA{160, 160, 160, 200}
	// Vertical tick pointing up
	vector.StrokeLine(screen, cx, baseY, cx, baseY-tickLen, 1, col, false)
	// Small arrowhead
	vector.StrokeLine(screen, cx-4, baseY-tickLen+5, cx, baseY-tickLen, 1, col, false)
	vector.StrokeLine(screen, cx+4, baseY-tickLen+5, cx, baseY-tickLen, 1, col, false)
	// "N" label
	ebitenutil.DebugPrintAt(screen, "N", int(cx)-3, int(baseY-tickLen)-13)
}

// GetDataTagFieldAt returns which data-tag field (ALTITUDE, SPEED, HEADING) lies under
// screen point (sx, sy), given the aircraft's screen-space position (ax, ay).
// Returns "" if the point is outside the tag or on row 1.
func (r *Renderer) GetDataTagFieldAt(a *aircraft.Aircraft, ax, ay, sx, sy float32) string {
	tagX := int(ax) + a.DataTagOffX
	tagY := int(ay) + a.DataTagOffY

	// Horizontal bounds
	if sx < float32(tagX-3) || sx > float32(tagX+133) {
		return ""
	}
	// Row 2 (altitude / speed): tagY+13 .. tagY+29
	if sy >= float32(tagY+13) && sy < float32(tagY+29) {
		if sx < float32(tagX+68) {
			return "ALTITUDE"
		}
		return "SPEED"
	}
	// Row 3 (heading): tagY+29 .. tagY+50
	if sy >= float32(tagY+29) && sy < float32(tagY+50) {
		return "HEADING"
	}
	return ""
}

// radarCenterX returns the horizontal pixel center of the radar area (accounting for panel offset).
func (r *Renderer) radarCenterX() int {
	return r.ScreenWidth/2 + r.RadarOffsetX
}

// worldToScreen converts world coordinates (nm) to screen coordinates
func (r *Renderer) worldToScreen(x, y float64) (float32, float32) {
	screenX := float32(r.radarCenterX()) + float32((x-r.CenterX)*r.Scale)
	screenY := float32(r.ScreenHeight/2) - float32((y-r.CenterY)*r.Scale)
	return screenX, screenY
}

// WorldToScreen converts world coordinates (nm) to screen coordinates (exported version)
func (r *Renderer) WorldToScreen(x, y float64) (float64, float64) {
	screenX := float64(r.radarCenterX()) + (x-r.CenterX)*r.Scale
	screenY := float64(r.ScreenHeight/2) - (y-r.CenterY)*r.Scale
	return screenX, screenY
}

// screenToWorld converts screen coordinates to world coordinates (nm)
func (r *Renderer) ScreenToWorld(screenX, screenY int) (float64, float64) {
	x := r.CenterX + float64(screenX-r.radarCenterX())/r.Scale
	y := r.CenterY - float64(screenY-r.ScreenHeight/2)/r.Scale
	return x, y
}
