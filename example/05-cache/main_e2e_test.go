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
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestGetItems(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/items/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var items []map[string]any
	json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestGetItemById(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/items/1", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var item map[string]any
	json.Unmarshal(w.Body.Bytes(), &item)
	if item["name"] != "Item" {
		t.Errorf("expected name 'Item', got %v", item["name"])
	}
}

func TestCacheConsistency(t *testing.T) {
	app := createTestApp(t)

	// First request
	req := httptest.NewRequest("GET", "/items/", nil)
	w1 := httptest.NewRecorder()
	app.Handler().ServeHTTP(w1, req)

	// Second request (should be served from cache)
	req = httptest.NewRequest("GET", "/items/", nil)
	w2 := httptest.NewRecorder()
	app.Handler().ServeHTTP(w2, req)

	if w1.Body.String() != w2.Body.String() {
		t.Error("expected identical responses from cache")
	}
}
