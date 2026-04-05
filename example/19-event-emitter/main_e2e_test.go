package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xfurai/gonest"
)

func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestCreateOrder(t *testing.T) {
	app := createTestApp(t)

	body := `{"name":"Widget","description":"A fine widget"}`
	req := httptest.NewRequest("POST", "/orders/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d; body: %s", w.Code, w.Body.String())
	}

	var order Order
	json.Unmarshal(w.Body.Bytes(), &order)
	if order.ID == 0 {
		t.Error("expected order ID > 0")
	}
	if order.Name != "Widget" {
		t.Errorf("expected name 'Widget', got %q", order.Name)
	}
}

func TestListOrders(t *testing.T) {
	app := createTestApp(t)

	// Create an order first
	body := `{"name":"Gadget","description":"A cool gadget"}`
	req := httptest.NewRequest("POST", "/orders/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	// List orders
	req = httptest.NewRequest("GET", "/orders/", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var orders []Order
	json.Unmarshal(w.Body.Bytes(), &orders)
	if len(orders) < 1 {
		t.Error("expected at least 1 order")
	}
}

func TestEventLog(t *testing.T) {
	app := createTestApp(t)

	// Create an order (triggers event)
	body := `{"name":"EventTest","description":"Testing events"}`
	req := httptest.NewRequest("POST", "/orders/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create failed: %d %s", w.Code, w.Body.String())
	}

	// Check event log
	req = httptest.NewRequest("GET", "/events/log", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var events []OrderCreatedEvent
	json.Unmarshal(w.Body.Bytes(), &events)
	if len(events) < 1 {
		t.Error("expected at least 1 event in log")
	}

	found := false
	for _, e := range events {
		if e.Name == "EventTest" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'EventTest' event in log")
	}
}

func TestEmptyOrdersList(t *testing.T) {
	// Fresh app — no orders yet
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/orders/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var orders []Order
	json.Unmarshal(w.Body.Bytes(), &orders)
	if orders == nil {
		// Empty slice should be [], not null
		var raw json.RawMessage
		json.Unmarshal(w.Body.Bytes(), &raw)
		if string(raw) == "null" {
			t.Error("expected empty array [], got null")
		}
	}
}
