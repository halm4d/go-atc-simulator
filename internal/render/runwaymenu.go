package render

import (
	"atc-sim/internal/airport"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	menuW      = 268
	menuTitleH = 20
	menuRowH   = 26
	menuBtnW   = 88
	menuBtnH   = 18
	menuPad    = 10
)

type menuButton struct {
	x, y, w, h int
	runway      string
	action      string // "LAND", "TAKEOFF", "CLOSE"
}

// RunwayMenu is a floating popup for selecting active landing and takeoff runways.
type RunwayMenu struct {
	Visible bool
	X, Y    int
	Airport *airport.Airport
	buttons []menuButton
}

// NewRunwayMenu creates a runway menu (hidden by default).
func NewRunwayMenu() *RunwayMenu {
	return &RunwayMenu{}
}

// Show displays the menu anchored near (x, y).
func (m *RunwayMenu) Show(x, y int, apt *airport.Airport, screenW, screenH int) {
	m.Airport = apt
	m.Visible = true
	h := m.totalHeight()
	// Clamp so it stays on screen
	if x+menuW > screenW {
		x = screenW - menuW - 4
	}
	if y+h > screenH {
		y = screenH - h - 4
	}
	m.X = x
	m.Y = y
}

// Hide closes the menu.
func (m *RunwayMenu) Hide() {
	m.Visible = false
}

func (m *RunwayMenu) totalHeight() int {
	if m.Airport == nil {
		return menuTitleH + menuPad*2
	}
	rows := len(m.Airport.Runways) * 2 // each runway has two ends
	return menuTitleH + menuPad + rows*menuRowH + menuPad
}

// Draw renders the popup. activeLanding and activeTakeoff are the currently selected runways.
func (m *RunwayMenu) Draw(screen *ebiten.Image, activeLanding, activeTakeoff string) {
	if !m.Visible || m.Airport == nil {
		return
	}

	h := m.totalHeight()
	m.buttons = m.buttons[:0]

	bgCol := color.RGBA{8, 18, 36, 245}
	borderCol := color.RGBA{0, 200, 255, 255}
	titleCol := color.RGBA{0, 90, 140, 255}

	// Background + border
	vector.DrawFilledRect(screen, float32(m.X), float32(m.Y), float32(menuW), float32(h), bgCol, false)
	vector.StrokeRect(screen, float32(m.X), float32(m.Y), float32(menuW), float32(h), 2, borderCol, false)

	// Title bar
	vector.DrawFilledRect(screen, float32(m.X), float32(m.Y), float32(menuW), float32(menuTitleH), titleCol, false)
	vector.StrokeLine(screen,
		float32(m.X), float32(m.Y+menuTitleH),
		float32(m.X+menuW), float32(m.Y+menuTitleH),
		1, borderCol, false)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("RUNWAY CONFIG  %s", m.Airport.ICAO), m.X+6, m.Y+4)

	// Close [X] button in title bar
	closeX := m.X + menuW - 26
	closeY := m.Y + 2
	vector.DrawFilledRect(screen, float32(closeX), float32(closeY), 22, float32(menuTitleH-4), color.RGBA{140, 0, 0, 200}, false)
	ebitenutil.DebugPrintAt(screen, " X", closeX, closeY+2)
	m.buttons = append(m.buttons, menuButton{x: closeX, y: closeY, w: 22, h: menuTitleH - 4, action: "CLOSE"})

	// Column header
	headerY := m.Y + menuTitleH + 2
	ebitenutil.DebugPrintAt(screen, "RWY", m.X+menuPad, headerY)
	ebitenutil.DebugPrintAt(screen, "LANDING", m.X+60, headerY)
	ebitenutil.DebugPrintAt(screen, "TAKEOFF", m.X+60+menuBtnW+6, headerY)

	rowY := m.Y + menuTitleH + menuPad + 8
	for _, rwy := range m.Airport.Runways {
		for _, name := range []string{rwy.Name1, rwy.Name2} {
			// Runway name
			ebitenutil.DebugPrintAt(screen, name, m.X+menuPad, rowY+4)

			// LAND button
			landX := m.X + 56
			isLand := name == activeLanding
			landBg := color.RGBA{20, 60, 20, 255}
			landBorder := color.RGBA{0, 180, 60, 180}
			if isLand {
				landBg = color.RGBA{0, 160, 50, 255}
				landBorder = color.RGBA{0, 255, 100, 255}
			}
			vector.DrawFilledRect(screen, float32(landX), float32(rowY), menuBtnW, menuBtnH, landBg, false)
			vector.StrokeRect(screen, float32(landX), float32(rowY), menuBtnW, menuBtnH, 1, landBorder, false)
			landLabel := "  LAND"
			if isLand {
				landLabel = "  LAND *"
			}
			ebitenutil.DebugPrintAt(screen, landLabel, landX+2, rowY+4)
			m.buttons = append(m.buttons, menuButton{x: landX, y: rowY, w: menuBtnW, h: menuBtnH, runway: name, action: "LAND"})

			// T/O button
			toX := landX + menuBtnW + 6
			isTO := name == activeTakeoff
			toBg := color.RGBA{60, 40, 0, 255}
			toBorder := color.RGBA{200, 140, 0, 180}
			if isTO {
				toBg = color.RGBA{180, 100, 0, 255}
				toBorder = color.RGBA{255, 200, 0, 255}
			}
			vector.DrawFilledRect(screen, float32(toX), float32(rowY), menuBtnW, menuBtnH, toBg, false)
			vector.StrokeRect(screen, float32(toX), float32(rowY), menuBtnW, menuBtnH, 1, toBorder, false)
			toLabel := "  T/O"
			if isTO {
				toLabel = "  T/O *"
			}
			ebitenutil.DebugPrintAt(screen, toLabel, toX+2, rowY+4)
			m.buttons = append(m.buttons, menuButton{x: toX, y: rowY, w: menuBtnW, h: menuBtnH, runway: name, action: "TAKEOFF"})

			rowY += menuRowH
		}
	}
}

// HandleClick checks if a button was clicked. Returns (runway, action).
// action is "LAND", "TAKEOFF", or "CLOSE". runway is "" for CLOSE.
func (m *RunwayMenu) HandleClick(mouseX, mouseY int) (runway, action string) {
	if !m.Visible {
		return "", ""
	}
	for _, btn := range m.buttons {
		if mouseX >= btn.x && mouseX <= btn.x+btn.w &&
			mouseY >= btn.y && mouseY <= btn.y+btn.h {
			return btn.runway, btn.action
		}
	}
	return "", ""
}

// IsMouseInside returns true if the mouse is inside the menu bounds.
func (m *RunwayMenu) IsMouseInside(mouseX, mouseY int) bool {
	if !m.Visible || m.Airport == nil {
		return false
	}
	h := m.totalHeight()
	return mouseX >= m.X && mouseX <= m.X+menuW &&
		mouseY >= m.Y && mouseY <= m.Y+h
}
