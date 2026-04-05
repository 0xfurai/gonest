package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0xfurai/gonest"
)

func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	// Apply the global throttle guard (set in main(), not in AppModule)
	app.UseGlobalGuards(gonest.NewThrottleGuard(100, time.Minute))
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestGetData(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["data"] != "hello" {
		t.Errorf("expected 'hello', got %q", body["data"])
	}
}

func TestExpensiveOperation(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("POST", "/api/expensive", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestThrottleExceeded(t *testing.T) {
	app := createTestApp(t)

	// Send 100 requests (within limit)
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/api/data", nil)
		req.RemoteAddr = "10.0.0.3:1234"
		w := httptest.NewRecorder()
		app.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	// 101st request should be throttled
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.RemoteAddr = "10.0.0.3:1234"
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}
