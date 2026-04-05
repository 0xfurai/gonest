package swagger

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gonest"
)

func TestSwaggerModule_Integration(t *testing.T) {
	gen := NewGenerator(Options{
		Title:   "Test API",
		Version: "1.0.0",
	})

	gen.AddRoute(RouteInfo{
		Method:  "GET",
		Path:    "/cats",
		Summary: "List cats",
		Tags:    []string{"cats"},
	})

	swaggerMod := Module(Options{
		Title:   "Test API",
		Version: "1.0.0",
		Path:    "/docs",
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{swaggerMod},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Test swagger UI
	req := httptest.NewRequest("GET", "/docs/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("swagger UI: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "swagger-ui") {
		t.Error("expected swagger UI HTML")
	}

	// Test swagger JSON
	req = httptest.NewRequest("GET", "/docs/json", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("swagger JSON: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Error("expected JSON content type")
	}
}
