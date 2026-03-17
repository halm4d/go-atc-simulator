package render

import (
	"atc-sim/internal/aircraft"
	"fmt"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	panelWidth    = 280
	panelTopY     = 48 // below top HUD bar
	panelBottomY  = 22 // above bottom HUD bar
	rowHeight     = 42
	headerHeight  = 22
	detailHeight  = 156
	commandHeight = 160
)

// FlightStripPanel displays all aircraft in a scrollable list with detail and command sections.
type FlightStripPanel struct {
	X, Y, Width, Height int
	ScrollOffset        int
	Visible             bool
}

// NewFlightStripPanel creates a new flight strip panel.
func NewFlightStripPanel(screenWidth, screenHeight int) *FlightStripPanel {
	p := &FlightStripPanel{
		Visible: true,
	}
	p.UpdateLayout(screenWidth, screenHeight)
	return p
}

// UpdateLayout recalculates panel position/size based on current screen dimensions.
func (p *FlightStripPanel) UpdateLayout(screenWidth, screenHeight int) {
	p.Width = panelWidth
	p.X = screenWidth - panelWidth
	p.Y = panelTopY
	p.Height = screenHeight - panelTopY - panelBottomY
}

// IsMouseInPanel checks if mouse coordinates are within the panel.
func (p *FlightStripPanel) IsMouseInPanel(mx, my int) bool {
	if !p.Visible {
		return false
	}
	return mx >= p.X && mx < p.X+p.Width && my >= p.Y && my < p.Y+p.Height
}

// listHeight returns the available pixel height for the aircraft list area.
func (p *FlightStripPanel) listHeight(hasSelection bool) int {
	h := p.Height - headerHeight
	if hasSelection {
		h -= detailHeight + commandHeight
	}
	if h < 0 {
		h = 0
	}
	return h
}

// HandleClick determines which aircraft was clicked in the list. Returns nil if no hit.
func (p *FlightStripPanel) HandleClick(mx, my int, aircraftList []*aircraft.Aircraft, selected *aircraft.Aircraft) *aircraft.Aircraft {
	if !p.Visible || !p.IsMouseInPanel(mx, my) {
		return nil
	}

	sorted := sortAircraft(aircraftList)
	listY := p.Y + headerHeight
	hasSelection := selected != nil
	listH := p.listHeight(hasSelection)

	relY := my - listY + p.ScrollOffset
	if relY < 0 || my < listY || my >= listY+listH {
		return nil
	}

	idx := relY / rowHeight
	if idx < 0 || idx >= len(sorted) {
		return nil
	}
	return sorted[idx]
}

// HandleScroll adjusts the scroll offset. Call with wheel delta.
func (p *FlightStripPanel) HandleScroll(wheelY float64, aircraftCount int, hasSelection bool) {
	p.ScrollOffset -= int(wheelY * 30)
	maxScroll := aircraftCount*rowHeight - p.listHeight(hasSelection)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.ScrollOffset > maxScroll {
		p.ScrollOffset = maxScroll
	}
	if p.ScrollOffset < 0 {
		p.ScrollOffset = 0
	}
}

// Draw renders the entire flight strip panel.
func (p *FlightStripPanel) Draw(screen *ebiten.Image, aircraftList []*aircraft.Aircraft,
	selected *aircraft.Aircraft, conflictMap map[*aircraft.Aircraft]string,
	commandMode, commandInput string) {
	if !p.Visible {
		return
	}

	// Panel background
	vector.FillRect(screen, float32(p.X), float32(p.Y),
		float32(p.Width), float32(p.Height),
		color.RGBA{6, 8, 6, 245}, false)

	// Panel border
	vector.StrokeRect(screen, float32(p.X), float32(p.Y),
		float32(p.Width), float32(p.Height),
		1, color.RGBA{0, 80, 0, 160}, false)

	// Header bar
	vector.FillRect(screen, float32(p.X), float32(p.Y),
		float32(p.Width), float32(headerHeight),
		color.RGBA{10, 20, 10, 230}, false)
	ebitenutil.DebugPrintAt(screen, "FLIGHT STRIPS", p.X+6, p.Y+5)
	countText := fmt.Sprintf("[%d]", len(aircraftList))
	ebitenutil.DebugPrintAt(screen, countText, p.X+p.Width-len(countText)*6-8, p.Y+5)

	// Sort aircraft for display
	sorted := sortAircraft(aircraftList)

	hasSelection := selected != nil
	listH := p.listHeight(hasSelection)
	listY := p.Y + headerHeight

	// Clamp scroll
	maxScroll := len(sorted)*rowHeight - listH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.ScrollOffset > maxScroll {
		p.ScrollOffset = maxScroll
	}
	if p.ScrollOffset < 0 {
		p.ScrollOffset = 0
	}

	// Draw aircraft rows (clipped to list area)
	for i, a := range sorted {
		rowY := listY + i*rowHeight - p.ScrollOffset
		// Skip rows outside visible area
		if rowY+rowHeight < listY || rowY >= listY+listH {
			continue
		}
		isSelected := selected != nil && a.Callsign == selected.Callsign
		severity := conflictMap[a]
		p.drawAircraftRow(screen, a, rowY, listY, listY+listH, isSelected, severity)
	}

	// Separator line below list
	sepY := float32(listY + listH)
	vector.StrokeLine(screen, float32(p.X), sepY, float32(p.X+p.Width), sepY,
		1, color.RGBA{0, 80, 0, 140}, false)

	// Scrollbar
	if len(sorted)*rowHeight > listH && listH > 0 {
		totalH := len(sorted) * rowHeight
		trackX := float32(p.X + p.Width - 4)
		trackH := float32(listH)
		thumbH := float32(listH) * float32(listH) / float32(totalH)
		if thumbH < 10 {
			thumbH = 10
		}
		thumbY := float32(listY) + float32(p.ScrollOffset)/float32(totalH)*trackH
		vector.FillRect(screen, trackX, float32(listY), 3, trackH,
			color.RGBA{20, 30, 20, 140}, false)
		vector.FillRect(screen, trackX, thumbY, 3, thumbH,
			color.RGBA{0, 140, 0, 180}, false)
	}

	// Detail and command sections (only when aircraft is selected)
	if hasSelection {
		detailY := listY + listH
		p.drawDetailSection(screen, selected, detailY)

		cmdY := detailY + detailHeight
		p.drawCommandSection(screen, selected, commandMode, commandInput, cmdY)
	}
}

// drawAircraftRow draws one aircraft entry in the list.
func (p *FlightStripPanel) drawAircraftRow(screen *ebiten.Image, a *aircraft.Aircraft,
	rowY, clipTop, clipBottom int, isSelected bool, conflictSeverity string) {

	col := AircraftColor(a, isSelected)

	// Row background tint
	bgAlpha := uint8(25)
	if isSelected {
		bgAlpha = 50
	}
	bgColor := color.RGBA{col.R / 4, col.G / 4, col.B / 4, bgAlpha}

	// Clamp drawing to visible area
	drawY := rowY
	drawH := rowHeight
	if drawY < clipTop {
		drawH -= clipTop - drawY
		drawY = clipTop
	}
	if drawY+drawH > clipBottom {
		drawH = clipBottom - drawY
	}
	if drawH <= 0 {
		return
	}

	vector.FillRect(screen, float32(p.X+1), float32(drawY),
		float32(p.Width-2), float32(drawH), bgColor, false)

	// Colored left bar indicator (shows aircraft phase color)
	vector.FillRect(screen, float32(p.X+1), float32(drawY),
		3, float32(drawH), col, false)

	// Selection indicator
	if isSelected {
		vector.StrokeRect(screen, float32(p.X+1), float32(rowY),
			float32(p.Width-2), float32(rowHeight), 1, ColorSelected, false)
	}

	// Conflict badge
	if conflictSeverity != "" {
		badgeCol := ColorConflict
		if conflictSeverity == "CRITICAL" {
			badgeCol = ColorCritical
		}
		vector.FillRect(screen, float32(p.X+p.Width-22), float32(rowY+2), 18, 12, badgeCol, false)
		ebitenutil.DebugPrintAt(screen, "CA", p.X+p.Width-20, rowY+2)
	}

	textX := p.X + 6

	// Row 1: Callsign, Type, Phase
	if rowY >= clipTop && rowY+14 <= clipBottom {
		phaseStr := phaseAbbrev(a)
		line1 := fmt.Sprintf("%-8s %-4s  %s", a.Callsign, a.Type.ICAO, phaseStr)
		drawColorText(screen, line1, textX, rowY+2, col)
	}

	// Row 2: Alt, Speed, Heading
	if rowY+14 >= clipTop && rowY+28 <= clipBottom {
		altStr := formatAlt(a.Altitude)
		tgtAltStr := formatAlt(a.TargetAltitude)
		altArrow := ""
		if a.TargetAltitude > a.Altitude+50 {
			altArrow = "↑"
		} else if a.TargetAltitude < a.Altitude-50 {
			altArrow = "↓"
		}
		spdStr := fmt.Sprintf("%d", int(a.Speed))
		hdgStr := fmt.Sprintf("%03d", int(a.Heading))

		var line2 string
		if a.Phase == aircraft.PhaseHoldingShort || a.Phase == aircraft.PhaseLineUpWait {
			line2 = fmt.Sprintf("RWY %-4s", a.RunwayName)
		} else {
			line2 = fmt.Sprintf("%s%s%s %skt H%s", altStr, altArrow, tgtAltStr, spdStr, hdgStr)
		}
		drawColorText(screen, line2, textX, rowY+15, dimColor(col, 0.75))
	}

	// Row 3: Route info
	if rowY+28 >= clipTop && rowY+42 <= clipBottom {
		line3 := routeInfo(a)
		drawColorText(screen, line3, textX, rowY+28, dimColor(col, 0.55))
	}

	// Bottom separator
	if rowY+rowHeight-1 >= clipTop && rowY+rowHeight-1 < clipBottom {
		vector.StrokeLine(screen, float32(p.X+4), float32(rowY+rowHeight-1),
			float32(p.X+p.Width-4), float32(rowY+rowHeight-1),
			1, color.RGBA{30, 40, 30, 80}, false)
	}
}

// drawDetailSection renders extended aircraft info when selected.
func (p *FlightStripPanel) drawDetailSection(screen *ebiten.Image, a *aircraft.Aircraft, y int) {
	// Background
	vector.FillRect(screen, float32(p.X), float32(y),
		float32(p.Width), float32(detailHeight),
		color.RGBA{4, 6, 4, 245}, false)

	// Header
	vector.FillRect(screen, float32(p.X), float32(y),
		float32(p.Width), 16,
		color.RGBA{8, 16, 8, 210}, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("DETAIL: %s", a.Callsign), p.X+6, y+3)

	x := p.X + 6
	ly := y + 20

	col := color.RGBA{180, 200, 180, 255}
	dimCol := color.RGBA{130, 150, 130, 255}

	// Type info
	drawColorText(screen, fmt.Sprintf("%s - %s", a.Type.ICAO, a.Type.Name), x, ly, col)
	ly += 14
	drawColorText(screen, fmt.Sprintf("Cat: %s  Wake: %s", a.Type.Category, a.Type.WakeTurbulence), x, ly, dimCol)
	ly += 16

	// Performance
	drawColorText(screen, fmt.Sprintf("Max SPD: %dkt  Min: %dkt", int(a.Type.MaxSpeed), int(a.Type.MinSpeed)), x, ly, col)
	ly += 14
	drawColorText(screen, fmt.Sprintf("Cruise: %dkt  Max ALT: FL%d", int(a.Type.CruiseSpeed), int(a.Type.MaxAltitude/100)), x, ly, col)
	ly += 14
	drawColorText(screen, fmt.Sprintf("Climb: %dfpm  Desc: %dfpm", int(a.Type.ClimbRate), int(a.Type.DescentRate)), x, ly, dimCol)
	ly += 14
	drawColorText(screen, fmt.Sprintf("Turn: %.1f deg/s", a.Type.TurnRate), x, ly, dimCol)
	ly += 16

	// Assignment
	phase := phaseFullName(a)
	cyanCol := color.RGBA{0, 180, 180, 255}
	if a.RunwayName != "" {
		drawColorText(screen, fmt.Sprintf("RWY: %s  Phase: %s", a.RunwayName, phase), x, ly, cyanCol)
	} else {
		drawColorText(screen, fmt.Sprintf("Phase: %s", phase), x, ly, cyanCol)
	}
	ly += 14

	// Assigned SID/STAR (preserved even after direct-to or heading commands)
	if a.AssignedRoute != "" {
		label := "STAR"
		if a.IsDeparture {
			label = "SID"
		}
		drawColorText(screen, fmt.Sprintf("Assigned %s: %s", label, a.AssignedRoute), x, ly, color.RGBA{200, 180, 0, 255})
	}
}

// drawCommandSection renders the command interface at the bottom of the panel.
func (p *FlightStripPanel) drawCommandSection(screen *ebiten.Image, a *aircraft.Aircraft,
	commandMode, commandInput string, y int) {

	// Background
	vector.FillRect(screen, float32(p.X), float32(y),
		float32(p.Width), float32(commandHeight),
		color.RGBA{4, 6, 4, 245}, false)

	// Header
	vector.FillRect(screen, float32(p.X), float32(y),
		float32(p.Width), 16,
		color.RGBA{6, 14, 6, 210}, false)
	ebitenutil.DebugPrintAt(screen, "COMMANDS", p.X+6, y+3)

	x := p.X + 6
	ly := y + 22

	if commandMode != "" {
		// Active command prompt
		prompt := fmt.Sprintf("%s > %s_", commandMode, commandInput)
		drawColorText(screen, prompt, x, ly, color.RGBA{255, 255, 0, 255})
		ly += 18

		var instruction string
		switch commandMode {
		case "HEADING":
			instruction = "Enter heading (0-359)"
		case "ALTITUDE":
			instruction = "Enter altitude (100s ft)"
		case "SPEED":
			instruction = "Enter speed (knots)"
		case "DIRECT":
			instruction = "Enter waypoint/route"
		}
		drawColorText(screen, instruction, x, ly, color.RGBA{150, 175, 150, 255})
		ly += 16
		drawColorText(screen, "ENTER=Confirm ESC=Cancel", x, ly, color.RGBA{100, 125, 100, 255})
	} else {
		// Show available commands based on phase
		cmdCol := color.RGBA{0, 200, 100, 255}
		dimCmdCol := color.RGBA{100, 150, 130, 255}

		switch a.Phase {
		case aircraft.PhaseHoldingShort:
			drawColorText(screen, fmt.Sprintf("RWY: %s", a.RunwayName), x, ly, cmdCol)
			ly += 15
			drawColorText(screen, "L - Line up and wait", x, ly, dimCmdCol)
			ly += 15
			drawColorText(screen, "T - Cleared for takeoff", x, ly, dimCmdCol)
		case aircraft.PhaseLineUpWait:
			drawColorText(screen, fmt.Sprintf("RWY: %s (lined up)", a.RunwayName), x, ly, cmdCol)
			ly += 15
			drawColorText(screen, "T - Cleared for takeoff", x, ly, dimCmdCol)
		case aircraft.PhaseTakeoffRoll:
			drawColorText(screen, "TAKEOFF ROLL...", x, ly, cmdCol)
		case aircraft.PhaseClimbout:
			drawColorText(screen, "[TOWER] Climbing out", x, ly, cmdCol)
			ly += 15
			drawColorText(screen, "H - Heading", x, ly, dimCmdCol)
		case aircraft.PhaseFinal:
			drawColorText(screen, fmt.Sprintf("ILS %s", a.RunwayName), x, ly, cmdCol)
			ly += 15
			drawColorText(screen, "C - Cleared to land", x, ly, dimCmdCol)
			ly += 15
			drawColorText(screen, "H/S - Heading/Speed", x, ly, dimCmdCol)
		case aircraft.PhaseLanding:
			drawColorText(screen, fmt.Sprintf("CLEARED %s", a.RunwayName), x, ly, cmdCol)
		case aircraft.PhaseHolding:
			drawColorText(screen, "HOLDING PATTERN", x, ly, cmdCol)
			ly += 15
			drawColorText(screen, "H - Exit hold (heading)", x, ly, dimCmdCol)
			ly += 15
			drawColorText(screen, "A - Altitude  S - Speed", x, ly, dimCmdCol)
		default:
			drawColorText(screen, "H - Heading", x, ly, dimCmdCol)
			ly += 15
			drawColorText(screen, "A - Altitude", x, ly, dimCmdCol)
			ly += 15
			drawColorText(screen, "S - Speed", x, ly, dimCmdCol)
			ly += 15
			drawColorText(screen, "D - Direct to waypoint", x, ly, dimCmdCol)
			ly += 15
			drawColorText(screen, "W - Enter hold", x, ly, dimCmdCol)
		}
	}
}

// --- Helper functions ---

func sortAircraft(list []*aircraft.Aircraft) []*aircraft.Aircraft {
	sorted := make([]*aircraft.Aircraft, len(list))
	copy(sorted, list)
	sort.Slice(sorted, func(i, j int) bool {
		pi := phasePriority(sorted[i])
		pj := phasePriority(sorted[j])
		if pi != pj {
			return pi < pj
		}
		return sorted[i].Callsign < sorted[j].Callsign
	})
	return sorted
}

func phasePriority(a *aircraft.Aircraft) int {
	switch a.Phase {
	case aircraft.PhaseFinal, aircraft.PhaseLanding:
		return 0 // Most urgent
	case aircraft.PhaseArrival:
		return 1
	case aircraft.PhaseHolding:
		return 2
	case aircraft.PhaseHoldingShort, aircraft.PhaseLineUpWait:
		return 3
	case aircraft.PhaseTakeoffRoll, aircraft.PhaseClimbout:
		return 4
	case aircraft.PhaseDeparture:
		return 5
	default:
		return 6
	}
}

func phaseAbbrev(a *aircraft.Aircraft) string {
	switch a.Phase {
	case aircraft.PhaseHoldingShort:
		return "GND"
	case aircraft.PhaseLineUpWait:
		return "L+W"
	case aircraft.PhaseTakeoffRoll:
		return "T/O"
	case aircraft.PhaseClimbout:
		return "CLB"
	case aircraft.PhaseDeparture:
		return "DEP"
	case aircraft.PhaseArrival:
		return "ARR"
	case aircraft.PhaseHolding:
		return "HLD"
	case aircraft.PhaseFinal:
		return "ILS"
	case aircraft.PhaseLanding:
		return "LND"
	default:
		return "???"
	}
}

func phaseFullName(a *aircraft.Aircraft) string {
	switch a.Phase {
	case aircraft.PhaseHoldingShort:
		return "Holding Short"
	case aircraft.PhaseLineUpWait:
		return "Line Up & Wait"
	case aircraft.PhaseTakeoffRoll:
		return "Takeoff Roll"
	case aircraft.PhaseClimbout:
		return "Climbout"
	case aircraft.PhaseDeparture:
		return "Departure"
	case aircraft.PhaseArrival:
		return "Arrival"
	case aircraft.PhaseHolding:
		return "Holding"
	case aircraft.PhaseFinal:
		return "ILS Approach"
	case aircraft.PhaseLanding:
		return "Landing"
	default:
		return "Unknown"
	}
}

func formatAlt(alt float64) string {
	return fmt.Sprintf("%03d", int(math.Round(alt/100)))
}

func routeInfo(a *aircraft.Aircraft) string {
	if a.Phase == aircraft.PhaseHoldingShort || a.Phase == aircraft.PhaseLineUpWait {
		if a.RouteName != "" {
			return fmt.Sprintf("SID: %s", a.RouteName)
		}
		return "Awaiting clearance"
	}
	if a.HasRoute && a.RouteName != "" {
		next := ""
		if len(a.RouteNames) > 0 {
			next = a.RouteNames[0]
		}
		if next != "" {
			return fmt.Sprintf("%s > %s", a.RouteName, next)
		}
		return a.RouteName
	}
	if a.DirectTarget != "" {
		return fmt.Sprintf("> %s", a.DirectTarget)
	}
	if a.Phase == aircraft.PhaseHolding {
		return fmt.Sprintf("HOLD HDG %03d", int(a.HoldInboundHeading))
	}
	return fmt.Sprintf("HDG %03d", int(a.Heading))
}

func drawColorText(screen *ebiten.Image, text string, x, y int, col color.RGBA) {
	// ebitenutil.DebugPrintAt draws in white; we draw a colored rect behind
	// Unfortunately DebugPrintAt only supports white text.
	// Workaround: use DebugPrintAt and tint by drawing colored overlay.
	// Actually, the simplest approach: just use DebugPrintAt for now (white text),
	// and use a colored indicator bar on the left edge of each row instead.
	// For a more polished look we'd need a font renderer.
	// Let's use DebugPrintAt which gives readable white text.
	ebitenutil.DebugPrintAt(screen, text, x, y)
}

func dimColor(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * factor),
		G: uint8(float64(c.G) * factor),
		B: uint8(float64(c.B) * factor),
		A: c.A,
	}
}
