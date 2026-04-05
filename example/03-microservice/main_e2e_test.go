package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/0xfurai/gonest"
)

var testApp *gonest.Application

func TestMain(m *testing.M) {
	// Start the TCP microservice before tests
	go startMicroservice()
	time.Sleep(200 * time.Millisecond)

	// Create the app (connects to TCP server)
	testApp = gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := testApp.Init(); err != nil {
		panic("init failed: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestSumEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/math/sum", nil)
	w := httptest.NewRecorder()
	testApp.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	sum, _ := body["sum"].(float64)
	if sum != 15 {
		t.Errorf("expected sum 15, got %v", body["sum"])
	}
}

func TestHelloEndpoint(t *testing.T) {
	req := httptest.NewRequest("GET", "/math/hello/World", nil)
	w := httptest.NewRecorder()
	testApp.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["greeting"] != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", body["greeting"])
	}
}
