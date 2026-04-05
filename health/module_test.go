package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xfurai/gonest"
)

func TestHealthService_AllUp(t *testing.T) {
	svc := NewHealthService()
	svc.AddIndicator(&PingIndicator{})
	svc.AddIndicator(&CustomIndicator{
		IndicatorName: "database",
		CheckFn: func() HealthResult {
			return HealthResult{Status: StatusUp, Details: map[string]any{"latency": "5ms"}}
		},
	})

	resp := svc.Check()
	if resp.Status != StatusUp {
		t.Errorf("expected 'up', got %q", resp.Status)
	}
	if len(resp.Info) != 2 {
		t.Errorf("expected 2 info entries, got %d", len(resp.Info))
	}
	if len(resp.Error) != 0 {
		t.Errorf("expected 0 errors, got %d", len(resp.Error))
	}
}

func TestHealthService_OneDown(t *testing.T) {
	svc := NewHealthService()
	svc.AddIndicator(&PingIndicator{})
	svc.AddIndicator(&CustomIndicator{
		IndicatorName: "redis",
		CheckFn: func() HealthResult {
			return HealthResult{Status: StatusDown, Details: map[string]any{"error": "connection refused"}}
		},
	})

	resp := svc.Check()
	if resp.Status != StatusDown {
		t.Errorf("expected 'down', got %q", resp.Status)
	}
	if len(resp.Error) != 1 {
		t.Errorf("expected 1 error, got %d", len(resp.Error))
	}
	if _, ok := resp.Error["redis"]; !ok {
		t.Error("expected redis in errors")
	}
}

func TestHealthService_NoIndicators(t *testing.T) {
	svc := NewHealthService()
	resp := svc.Check()
	if resp.Status != StatusUp {
		t.Errorf("expected 'up' with no indicators, got %q", resp.Status)
	}
}

func TestHealthModule_Integration(t *testing.T) {
	healthMod := NewModule(Options{
		Path: "/health",
		Indicators: []HealthIndicator{
			&PingIndicator{},
			&CustomIndicator{
				IndicatorName: "app",
				CheckFn: func() HealthResult {
					return HealthResult{Status: StatusUp}
				},
			},
		},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{healthMod},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp HealthResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Status != StatusUp {
		t.Errorf("expected 'up', got %q", resp.Status)
	}
	if len(resp.Details) != 2 {
		t.Errorf("expected 2 details, got %d", len(resp.Details))
	}
}

func TestHealthModule_ServiceUnavailable(t *testing.T) {
	healthMod := NewModule(Options{
		Indicators: []HealthIndicator{
			&CustomIndicator{
				IndicatorName: "db",
				CheckFn: func() HealthResult {
					return HealthResult{Status: StatusDown}
				},
			},
		},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{healthMod},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.Init()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestPingIndicator(t *testing.T) {
	p := &PingIndicator{}
	if p.Name() != "ping" {
		t.Errorf("expected 'ping', got %q", p.Name())
	}
	r := p.Check()
	if r.Status != StatusUp {
		t.Error("expected up")
	}
}
