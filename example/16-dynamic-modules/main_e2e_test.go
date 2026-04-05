package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xfurai/gonest"
)

func TestDynamicModuleDefaultConfig(t *testing.T) {
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "Hello from dynamic module!" {
		t.Errorf("expected default greeting, got %q", body["message"])
	}
}

func TestDynamicModuleCustomConfig(t *testing.T) {
	// Create module with custom greeting
	customModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{GreetingModuleForRoot(GreetingOptions{Message: "Custom greeting!"})},
		Controllers: []any{NewAppController},
	})

	app := gonest.Create(customModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "Custom greeting!" {
		t.Errorf("expected custom greeting, got %q", body["message"])
	}
}
