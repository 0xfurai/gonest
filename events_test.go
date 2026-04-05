package gonest

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestEventEmitter_OnAndEmit(t *testing.T) {
	emitter := NewEventEmitter()
	var received string

	emitter.On("user.created", func(data any) error {
		received = data.(string)
		return nil
	})

	err := emitter.Emit("user.created", "john")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received != "john" {
		t.Errorf("expected 'john', got %q", received)
	}
}

func TestEventEmitter_MultipleHandlers(t *testing.T) {
	emitter := NewEventEmitter()
	var count int

	emitter.On("event", func(data any) error { count++; return nil })
	emitter.On("event", func(data any) error { count++; return nil })
	emitter.On("event", func(data any) error { count++; return nil })

	emitter.Emit("event", nil)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestEventEmitter_NoHandlers(t *testing.T) {
	emitter := NewEventEmitter()
	err := emitter.Emit("nonexistent", nil)
	if err != nil {
		t.Errorf("expected no error for missing event, got %v", err)
	}
}

func TestEventEmitter_EmitAsync(t *testing.T) {
	emitter := NewEventEmitter()
	var called atomic.Int32

	emitter.On("async", func(data any) error {
		called.Add(1)
		return nil
	})

	emitter.EmitAsync("async", nil)
	time.Sleep(50 * time.Millisecond)

	if called.Load() != 1 {
		t.Errorf("expected 1, got %d", called.Load())
	}
}

func TestEventEmitter_RemoveAll(t *testing.T) {
	emitter := NewEventEmitter()
	emitter.On("event", func(data any) error { return nil })
	emitter.On("event", func(data any) error { return nil })

	if emitter.ListenerCount("event") != 2 {
		t.Errorf("expected 2 listeners, got %d", emitter.ListenerCount("event"))
	}

	emitter.RemoveAll("event")
	if emitter.ListenerCount("event") != 0 {
		t.Errorf("expected 0 listeners after remove, got %d", emitter.ListenerCount("event"))
	}
}

func TestEventEmitter_ListenerCount(t *testing.T) {
	emitter := NewEventEmitter()
	if emitter.ListenerCount("missing") != 0 {
		t.Error("expected 0 for non-existent event")
	}

	emitter.On("test", func(data any) error { return nil })
	if emitter.ListenerCount("test") != 1 {
		t.Errorf("expected 1, got %d", emitter.ListenerCount("test"))
	}
}

func TestEventEmitter_HandlerError(t *testing.T) {
	emitter := NewEventEmitter()
	emitter.On("fail", func(data any) error {
		return NewInternalServerError("handler failed")
	})

	err := emitter.Emit("fail", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
