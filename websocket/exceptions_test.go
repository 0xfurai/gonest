package websocket

import (
	"errors"
	"testing"
)

func TestWsException(t *testing.T) {
	ex := NewWsException("ws error")
	if ex.Error() != "ws error" {
		t.Errorf("expected 'ws error', got %q", ex.Error())
	}
	if ex.Cause() != nil {
		t.Error("expected nil cause")
	}
}

func TestWsException_Wrapped(t *testing.T) {
	cause := errors.New("root cause")
	ex := WrapWsException("wrapped", cause)
	if ex.Error() != "wrapped: root cause" {
		t.Errorf("expected 'wrapped: root cause', got %q", ex.Error())
	}
	if !errors.Is(ex, cause) {
		t.Error("expected errors.Is to find cause")
	}
}
