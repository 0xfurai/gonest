package gonest

import (
	"testing"
)

func TestSSEStream_SendAndClose(t *testing.T) {
	stream := NewSSEStream(5)

	stream.Send(SSEEvent{Event: "update", Data: "hello"})
	stream.Send(SSEEvent{ID: "1", Data: map[string]int{"count": 42}})

	// Read events
	e1 := <-stream.events
	if e1.Event != "update" {
		t.Errorf("expected 'update', got %q", e1.Event)
	}

	e2 := <-stream.events
	if e2.ID != "1" {
		t.Errorf("expected id '1', got %q", e2.ID)
	}

	stream.Close()
}

func TestSSEStream_SendAfterClose(t *testing.T) {
	stream := NewSSEStream(1)
	stream.Close()

	// Should not block or panic
	stream.Send(SSEEvent{Data: "should be dropped"})
}

func TestSSEEvent_Fields(t *testing.T) {
	e := SSEEvent{
		ID:    "42",
		Event: "message",
		Data:  "hello world",
	}
	if e.ID != "42" {
		t.Errorf("expected '42', got %q", e.ID)
	}
	if e.Event != "message" {
		t.Errorf("expected 'message', got %q", e.Event)
	}
	if e.Data != "hello world" {
		t.Errorf("expected 'hello world', got %v", e.Data)
	}
}
