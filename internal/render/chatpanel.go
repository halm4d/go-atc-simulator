package render

import (
	"atc-sim/internal/chat"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ChatPanel is the draggable chat UI overlay.
type ChatPanel struct {
	X, Y          int
	Width, Height int
	Focused       bool
	InputText     string
	ScrollOffset  int // Number of messages scrolled up from bottom

	// Drag state
	IsDragging  bool
	DragOffsetX int
	DragOffsetY int

	titleBarH   int
	inputBarH   int
	lineH       int
	bgColor     color.RGBA
	borderColor color.RGBA
}

// NewChatPanel creates a chat panel at the bottom of the screen.
func NewChatPanel(screenW, screenH int) *ChatPanel {
	w := screenW - 40
	h := 180
	return &ChatPanel{
		X:           20,
		Y:           screenH - h - 10,
		Width:       w,
		Height:      h,
		titleBarH:   18,
		inputBarH:   22,
		lineH:       14,
		bgColor:     color.RGBA{6, 8, 6, 220},
		borderColor: color.RGBA{0, 120, 0, 180},
	}
}

// UpdateLayout repositions the panel when the screen resizes.
func (cp *ChatPanel) UpdateLayout(screenW, screenH int) {
	if !cp.IsDragging {
		cp.Width = screenW - 40
	}
}

// HandleInput processes keyboard and mouse input for the chat panel.
// Returns the submitted text (non-empty) when user presses Enter, or "" otherwise.
func (cp *ChatPanel) HandleInput(mouseX, mouseY int) string {
	// Handle drag
	cp.handleDrag(mouseX, mouseY)

	// Focus on click inside input bar or '/' key
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		inInputBar := mouseX >= cp.X && mouseX <= cp.X+cp.Width &&
			mouseY >= cp.Y+cp.Height-cp.inputBarH && mouseY <= cp.Y+cp.Height
		if inInputBar {
			cp.Focused = true
		} else {
			// Check if click is outside the panel entirely
			if mouseX < cp.X || mouseX > cp.X+cp.Width || mouseY < cp.Y || mouseY > cp.Y+cp.Height {
				cp.Focused = false
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySlash) && !cp.Focused {
		cp.Focused = true
		return ""
	}

	if !cp.Focused {
		return ""
	}

	// ESC unfocuses
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		cp.Focused = false
		return ""
	}

	// Enter submits
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		text := strings.TrimSpace(cp.InputText)
		cp.InputText = ""
		cp.ScrollOffset = 0
		return text
	}

	// Backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(cp.InputText) > 0 {
		cp.InputText = cp.InputText[:len(cp.InputText)-1]
		return ""
	}

	// Collect typed characters
	cp.InputText += string(ebiten.AppendInputChars(nil))

	// Scroll with mouse wheel when hovering over message area
	if mouseX >= cp.X && mouseX <= cp.X+cp.Width && mouseY >= cp.Y && mouseY < cp.Y+cp.Height-cp.inputBarH {
		_, wy := ebiten.Wheel()
		if wy > 0 {
			cp.ScrollOffset++
		} else if wy < 0 && cp.ScrollOffset > 0 {
			cp.ScrollOffset--
		}
	}

	return ""
}

func (cp *ChatPanel) handleDrag(mouseX, mouseY int) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		// Check if click is in the title bar
		if mouseX >= cp.X && mouseX <= cp.X+cp.Width &&
			mouseY >= cp.Y && mouseY <= cp.Y+cp.titleBarH {
			cp.IsDragging = true
			cp.DragOffsetX = mouseX - cp.X
			cp.DragOffsetY = mouseY - cp.Y
		}
	}
	if cp.IsDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			cp.X = mouseX - cp.DragOffsetX
			cp.Y = mouseY - cp.DragOffsetY
		} else {
			cp.IsDragging = false
		}
	}
}

// Draw renders the chat panel.
func (cp *ChatPanel) Draw(screen *ebiten.Image, history *chat.History) {
	x := float32(cp.X)
	y := float32(cp.Y)
	w := float32(cp.Width)
	h := float32(cp.Height)

	// Background
	vector.FillRect(screen, x, y, w, h, cp.bgColor, false)
	vector.StrokeRect(screen, x, y, w, h, 1, cp.borderColor, false)

	// Title bar
	vector.FillRect(screen, x, y, w, float32(cp.titleBarH), color.RGBA{0, 60, 0, 200}, false)
	ebitenutil.DebugPrintAt(screen, "RADIO", cp.X+4, cp.Y+3)

	// Message area
	msgAreaTop := cp.Y + cp.titleBarH
	msgAreaBottom := cp.Y + cp.Height - cp.inputBarH
	visibleLines := (msgAreaBottom - msgAreaTop) / cp.lineH

	if history != nil && len(history.Messages) > 0 {
		msgs := history.Messages
		// Apply scroll offset
		end := len(msgs) - cp.ScrollOffset
		if end < 0 {
			end = 0
		}
		start := end - visibleLines
		if start < 0 {
			start = 0
		}

		drawY := msgAreaTop + 2
		for i := start; i < end; i++ {
			msg := msgs[i]
			line := msg.Prefix() + ": " + msg.Text
			// Truncate if too long
			maxChars := (cp.Width - 10) / 6 // ~6px per char at debug font size
			if len(line) > maxChars {
				line = line[:maxChars-3] + "..."
			}
			ebitenutil.DebugPrintAt(screen, line, cp.X+4, drawY)
			drawY += cp.lineH
		}
	}

	// Input bar
	inputY := float32(cp.Y + cp.Height - cp.inputBarH)
	vector.FillRect(screen, x, inputY, w, float32(cp.inputBarH), color.RGBA{3, 5, 3, 240}, false)
	vector.StrokeLine(screen, x, inputY, x+w, inputY, 1, cp.borderColor, false)

	prompt := "> " + cp.InputText
	if cp.Focused {
		// Blinking cursor
		prompt += "_"
	}
	ebitenutil.DebugPrintAt(screen, prompt, cp.X+4, int(inputY)+4)
}

// IsFocused returns whether the chat input bar has focus.
func (cp *ChatPanel) IsFocused() bool {
	return cp.Focused
}
