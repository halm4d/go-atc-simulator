package game

import (
	"atc-sim/internal/aircraft"
	"atc-sim/internal/render"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// InputHandler handles user input
type InputHandler struct {
	MouseX int
	MouseY int
}

// NewInputHandler creates a new input handler
func NewInputHandler() *InputHandler {
	return &InputHandler{}
}

// Update updates input state
func (ih *InputHandler) Update() {
	ih.MouseX, ih.MouseY = ebiten.CursorPosition()
}

// HandleMouseClick checks if any aircraft was clicked (symbol or datatag)
func (ih *InputHandler) HandleMouseClick(aircraftList []*aircraft.Aircraft, worldX, worldY float64, renderer *render.Renderer) (*aircraft.Aircraft, bool) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// First check datatag clicks (higher priority)
		for _, a := range aircraftList {
			screenX, screenY := renderer.WorldToScreen(a.X, a.Y)
			x1, y1, x2, y2 := renderer.GetDataTagBounds(a, float32(screenX), float32(screenY))

			// Check if mouse is within datatag bounds
			if ih.MouseX >= x1 && ih.MouseX <= x2 && ih.MouseY >= y1 && ih.MouseY <= y2 {
				return a, true // Return aircraft and true to indicate datatag click
			}
		}

		// Then check aircraft symbol clicks
		for _, a := range aircraftList {
			// Calculate distance from click to aircraft
			dx := a.X - worldX
			dy := a.Y - worldY
			distNm := math.Sqrt(dx*dx + dy*dy)

			// If within 1nm, select it
			if distNm < 1.0 {
				return a, false // Return aircraft and false for symbol click
			}
		}
	}
	return nil, false
}

// IsKeyJustPressed checks if a key was just pressed
func (ih *InputHandler) IsKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

// GetNumberInput gets numeric input from keyboard
func (ih *InputHandler) GetNumberInput() (int, bool) {
	keys := []ebiten.Key{
		ebiten.Key0, ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4,
		ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9,
	}

	for i, key := range keys {
		if inpututil.IsKeyJustPressed(key) {
			return i, true
		}
	}
	return 0, false
}
