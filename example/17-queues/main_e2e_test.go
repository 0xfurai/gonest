package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestTranscodeJob(t *testing.T) {
	app := createTestApp(t)

	body := `{"file":"test.mp3"}`
	req := httptest.NewRequest("POST", "/audio/transcode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["jobId"] == "" {
		t.Error("expected jobId in response")
	}
	if resp["status"] != "queued" {
		t.Errorf("expected status 'queued', got %q", resp["status"])
	}
}

func TestQueueStatus(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/audio/status", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if _, ok := resp["pending"]; !ok {
		t.Error("expected 'pending' in response")
	}
	if _, ok := resp["processed"]; !ok {
		t.Error("expected 'processed' in response")
	}
}

func TestJobProcessing(t *testing.T) {
	app := createTestApp(t)

	// Enqueue a job
	body := `{"file":"process-test.mp3"}`
	req := httptest.NewRequest("POST", "/audio/transcode", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}

	// Wait briefly for the worker to process
	time.Sleep(100 * time.Millisecond)

	// Check status — processed count should have increased
	req = httptest.NewRequest("GET", "/audio/status", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	processed, _ := resp["processed"].(float64)
	if processed < 1 {
		t.Errorf("expected at least 1 processed job, got %v", resp["processed"])
	}
}
