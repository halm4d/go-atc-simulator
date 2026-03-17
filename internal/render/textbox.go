package render

import (
	"atc-sim/internal/aircraft"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// CommandTextBox represents a draggable command input textbox
type CommandTextBox struct {
	X               int
	Y               int
	Width           int
	Height          int
	IsDragging      bool
	DragOffsetX     int
	DragOffsetY     int
	BackgroundColor color.RGBA
	BorderColor     color.RGBA
	TextColor       color.RGBA
	TitleBarHeight  int
}

// NewCommandTextBox creates a new command textbox
func NewCommandTextBox(x, y, width, height int) *CommandTextBox {
	return &CommandTextBox{
		X:               x,
		Y:               y,
		Width:           width,
		Height:          height,
		IsDragging:      false,
		BackgroundColor: color.RGBA{6, 8, 6, 235},     // Near-black, semi-transparent
		BorderColor:     color.RGBA{0, 120, 0, 200},   // Dim green border
		TextColor:       color.RGBA{0, 220, 80, 255},  // Phosphor green text
		TitleBarHeight:  20,
	}
}

// Draw renders the command textbox
func (ctb *CommandTextBox) Draw(screen *ebiten.Image, commandMode, commandInput string, selectedAircraft *aircraft.Aircraft) {
	// Draw background
	vector.FillRect(screen,
		float32(ctb.X), float32(ctb.Y),
		float32(ctb.Width), float32(ctb.Height),
		ctb.BackgroundColor, false)

	// Draw title bar (draggable area)
	titleBarColor := color.RGBA{10, 20, 10, 255} // Very dark green title bar
	if ctb.IsDragging {
		titleBarColor = color.RGBA{20, 40, 20, 255} // Slightly brighter when dragging
	}
	vector.FillRect(screen,
		float32(ctb.X), float32(ctb.Y),
		float32(ctb.Width), float32(ctb.TitleBarHeight),
		titleBarColor, false)

	// Draw border
	vector.StrokeRect(screen,
		float32(ctb.X), float32(ctb.Y),
		float32(ctb.Width), float32(ctb.Height),
		2, ctb.BorderColor, false)

	// Draw title bar separator
	vector.StrokeLine(screen,
		float32(ctb.X), float32(ctb.Y+ctb.TitleBarHeight),
		float32(ctb.X+ctb.Width), float32(ctb.Y+ctb.TitleBarHeight),
		1, ctb.BorderColor, false)

	// Draw title
	ebitenutil.DebugPrintAt(screen, "COMMAND INPUT", ctb.X+5, ctb.Y+5)

	// Draw selected aircraft info
	contentY := ctb.Y + ctb.TitleBarHeight + 10
	if selectedAircraft != nil {
		text := fmt.Sprintf("Aircraft: %s", selectedAircraft.Callsign)
		ebitenutil.DebugPrintAt(screen, text, ctb.X+10, contentY)
		contentY += 20
	} else {
		ebitenutil.DebugPrintAt(screen, "No aircraft selected", ctb.X+10, contentY)
		contentY += 20
	}

	// Draw command mode and input
	if commandMode != "" {
		// Command prompt
		prompt := fmt.Sprintf("%s > %s_", commandMode, commandInput)
		ebitenutil.DebugPrintAt(screen, prompt, ctb.X+10, contentY)
		contentY += 20

		var instruction string
		switch commandMode {
		case "HEADING":
			instruction = "Enter heading (0-359)"
		case "ALTITUDE":
			instruction = "Enter altitude (in 100s ft)"
		case "SPEED":
			instruction = "Enter speed (knots)"
		case "DIRECT":
			instruction = "Enter waypoint name"
		}
		ebitenutil.DebugPrintAt(screen, instruction, ctb.X+10, contentY)
		contentY += 20
		ebitenutil.DebugPrintAt(screen, "ENTER to confirm", ctb.X+10, contentY)
		contentY += 15
		ebitenutil.DebugPrintAt(screen, "ESC to cancel", ctb.X+10, contentY)
	} else if selectedAircraft != nil {
		ebitenutil.DebugPrintAt(screen, "Commands:", ctb.X+10, contentY)
		contentY += 20

		switch selectedAircraft.Phase {
		case aircraft.PhaseHoldingShort:
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("RWY: %s", selectedAircraft.RunwayName), ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "L - Line up and wait", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "T - Cleared for takeoff", ctb.X+10, contentY)
		case aircraft.PhaseLineUpWait:
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("RWY: %s (lined up)", selectedAircraft.RunwayName), ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "T - Cleared for takeoff", ctb.X+10, contentY)
		case aircraft.PhaseTakeoffRoll:
			ebitenutil.DebugPrintAt(screen, "TAKEOFF ROLL...", ctb.X+10, contentY)
		case aircraft.PhaseClimbout:
			ebitenutil.DebugPrintAt(screen, "[TOWER] Climbing out", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "H - Heading", ctb.X+10, contentY)
		case aircraft.PhaseFinal:
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("ILS %s", selectedAircraft.RunwayName), ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "C - Cleared to land", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "H/S - Heading/Speed", ctb.X+10, contentY)
		case aircraft.PhaseLanding:
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("CLEARED %s", selectedAircraft.RunwayName), ctb.X+10, contentY)
		case aircraft.PhaseHolding:
			ebitenutil.DebugPrintAt(screen, "HOLDING PATTERN", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "H - Exit hold (new hdg)", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "A - Altitude", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "S - Speed", ctb.X+10, contentY)
		default:
			ebitenutil.DebugPrintAt(screen, "H - Heading", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "A - Altitude", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "S - Speed", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "D - Direct to waypoint", ctb.X+10, contentY)
			contentY += 15
			ebitenutil.DebugPrintAt(screen, "W - Enter hold", ctb.X+10, contentY)
		}
	}
}

// IsMouseInTitleBar checks if mouse is in the draggable title bar area
func (ctb *CommandTextBox) IsMouseInTitleBar(mouseX, mouseY int) bool {
	return mouseX >= ctb.X && mouseX <= ctb.X+ctb.Width &&
		mouseY >= ctb.Y && mouseY <= ctb.Y+ctb.TitleBarHeight
}

// IsMouseInTextBox checks if mouse is anywhere in the textbox
func (ctb *CommandTextBox) IsMouseInTextBox(mouseX, mouseY int) bool {
	return mouseX >= ctb.X && mouseX <= ctb.X+ctb.Width &&
		mouseY >= ctb.Y && mouseY <= ctb.Y+ctb.Height
}

// StartDrag starts dragging the textbox
func (ctb *CommandTextBox) StartDrag(mouseX, mouseY int) {
	ctb.IsDragging = true
	ctb.DragOffsetX = mouseX - ctb.X
	ctb.DragOffsetY = mouseY - ctb.Y
}

// UpdateDrag updates the textbox position while dragging
func (ctb *CommandTextBox) UpdateDrag(mouseX, mouseY int, screenWidth, screenHeight int) {
	if ctb.IsDragging {
		ctb.X = mouseX - ctb.DragOffsetX
		ctb.Y = mouseY - ctb.DragOffsetY

		// Keep within screen bounds
		if ctb.X < 0 {
			ctb.X = 0
		}
		if ctb.Y < 0 {
			ctb.Y = 0
		}
		if ctb.X+ctb.Width > screenWidth {
			ctb.X = screenWidth - ctb.Width
		}
		if ctb.Y+ctb.Height > screenHeight {
			ctb.Y = screenHeight - ctb.Height
		}
	}
}

// StopDrag stops dragging the textbox
func (ctb *CommandTextBox) StopDrag() {
	ctb.IsDragging = false
}
