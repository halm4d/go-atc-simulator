package chat

import "testing"

func TestHistoryAdd(t *testing.T) {
	h := NewHistory(3)
	h.Add(NewMessage(MsgATC, "ATC", "test1"))
	h.Add(NewMessage(MsgPilotReadback, "WZZ123", "test2"))
	if len(h.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(h.Messages))
	}
}

func TestHistoryMaxSize(t *testing.T) {
	h := NewHistory(3)
	h.Add(NewMessage(MsgATC, "ATC", "msg1"))
	h.Add(NewMessage(MsgATC, "ATC", "msg2"))
	h.Add(NewMessage(MsgATC, "ATC", "msg3"))
	h.Add(NewMessage(MsgATC, "ATC", "msg4"))
	if len(h.Messages) != 3 {
		t.Errorf("expected 3 messages (max), got %d", len(h.Messages))
	}
	if h.Messages[0].Text != "msg2" {
		t.Errorf("expected oldest message 'msg2', got %q", h.Messages[0].Text)
	}
}

func TestMessageTypes(t *testing.T) {
	m := NewMessage(MsgPilotRequest, "RYR456", "ready for departure")
	if m.Type != MsgPilotRequest {
		t.Errorf("wrong type: %d", m.Type)
	}
	if m.Sender != "RYR456" {
		t.Errorf("wrong sender: %s", m.Sender)
	}
}
