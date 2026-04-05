package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonest"
	"github.com/gonest/health"
)

// createTestApp builds the module from scratch since main() creates it locally.
func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()

	healthMod := health.NewModule(health.Options{
		Indicators: []health.HealthIndicator{
			&health.PingIndicator{},
			&health.CustomIndicator{
				IndicatorName: "app",
				CheckFn: func() health.HealthResult {
					return health.HealthResult{
						Status:  health.StatusUp,
						Details: map[string]any{"version": "1.0.0"},
					}
				},
			},
		},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{healthMod},
		Controllers: []any{NewAppController},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestHealthCheck(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "up" {
		t.Errorf("expected status 'up', got %v", body["status"])
	}
}

func TestRootEndpoint(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "App running" {
		t.Errorf("expected 'App running', got %q", body["message"])
	}
}
