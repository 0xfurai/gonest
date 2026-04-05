package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0xfurai/gonest"
)

func TestSSEEndpoint(t *testing.T) {
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Use a short-lived context since the SSE handler blocks
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req := httptest.NewRequest("GET", "/sse", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		app.Handler().ServeHTTP(w, req)
		close(done)
	}()

	// Wait for the handler goroutine to finish (it returns when context is cancelled)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("SSE handler didn't return within timeout")
	}

	// Verify SSE content type was set
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("expected Content-Type 'text/event-stream', got %q", ct)
	}

	// Verify at least some SSE data was written
	body := w.Body.String()
	if len(body) > 0 && !strings.Contains(body, "data:") {
		t.Errorf("expected SSE event data in body, got %q", body)
	}
}
