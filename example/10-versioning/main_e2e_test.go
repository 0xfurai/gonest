package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonest"
)

func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type:           gonest.VersioningURI,
		DefaultVersion: "1",
	}))
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestV1Cats(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/v1/cats/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["version"] != "1" {
		t.Errorf("expected version '1', got %v", body["version"])
	}
}

func TestV2Cats(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/v2/cats/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["version"] != "2" {
		t.Errorf("expected version '2', got %v", body["version"])
	}
	if body["data"] == nil {
		t.Error("expected 'data' field in v2 response")
	}
}
