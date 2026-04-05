package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xfurai/gonest"
)

func TestConfigEndpoint(t *testing.T) {
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/config", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)

	// Verify default values are present
	if body["port"] != "3000" {
		t.Errorf("expected port '3000', got %v", body["port"])
	}
	if body["env"] != "development" {
		t.Errorf("expected env 'development', got %v", body["env"])
	}
	if body["db_host"] != "localhost" {
		t.Errorf("expected db_host 'localhost', got %v", body["db_host"])
	}
	if body["debug"] != false {
		t.Errorf("expected debug false, got %v", body["debug"])
	}
}
