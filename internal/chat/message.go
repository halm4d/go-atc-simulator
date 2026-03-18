package chat

import "image/color"

// MessageType identifies the kind of chat message.
type MessageType int

const (
	MsgATC          MessageType = iota // Player's ATC command
	MsgPilotReadback                   // Pilot reading back a command
	MsgPilotRequest                    // Pilot-initiated request
	MsgSystem                          // System/error message
)

// Message is a single chat entry.
type Message struct {
	Type   MessageType
	Sender string // "ATC", callsign, or "SYS"
	Text   string
}

// NewMessage creates a chat message.
func NewMessage(t MessageType, sender, text string) Message {
	return Message{Type: t, Sender: sender, Text: text}
}

// Color returns the display color for this message type.
func (m Message) Color() color.RGBA {
	switch m.Type {
	case MsgATC:
		return color.RGBA{0, 200, 70, 255} // Green
	case MsgPilotReadback:
		return color.RGBA{0, 180, 220, 255} // Cyan
	case MsgPilotRequest:
		return color.RGBA{220, 160, 0, 255} // Amber
	case MsgSystem:
		return color.RGBA{255, 48, 48, 255} // Red
	}
	return color.RGBA{160, 165, 160, 255}
}

// Prefix returns the display prefix for this message.
func (m Message) Prefix() string {
	switch m.Type {
	case MsgATC:
		return "ATC"
	case MsgSystem:
		return "SYS"
	default:
		return m.Sender
	}
}

// History is a bounded FIFO buffer of chat messages.
type History struct {
	Messages []Message
	MaxSize  int
}

// NewHistory creates a history with the given capacity.
func NewHistory(maxSize int) *History {
	return &History{
		Messages: make([]Message, 0, maxSize),
		MaxSize:  maxSize,
	}
}

// Add appends a message, trimming the oldest if at capacity.
func (h *History) Add(m Message) {
	h.Messages = append(h.Messages, m)
	if len(h.Messages) > h.MaxSize {
		h.Messages = h.Messages[1:]
	}
}
