package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xfurai/gonest"
)

func TestAPIEndpoint(t *testing.T) {
	app := gonest.Create(AppModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "Hello from GoNest API!" {
		t.Errorf("expected API message, got %q", body["message"])
	}
}

func TestStaticFileServing(t *testing.T) {
	// Create temp directory with a test file
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("static content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Build a module pointing to the temp dir
	staticCtrl := &staticControllerWithDir{dir: dir}
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{
			NewAppController,
			func() *staticControllerWithDir { return staticCtrl },
		},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/static/hello.txt", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "static content" {
		t.Errorf("expected 'static content', got %q", w.Body.String())
	}
}

func TestStaticFileNotFound(t *testing.T) {
	dir := t.TempDir()
	staticCtrl := &staticControllerWithDir{dir: dir}
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *staticControllerWithDir { return staticCtrl }},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/static/nonexistent.txt", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// staticControllerWithDir allows pointing to a custom directory for tests.
type staticControllerWithDir struct {
	dir string
}

func (c *staticControllerWithDir) Register(r gonest.Router) {
	r.Get("/static/*", gonest.StaticFiles("/static/", c.dir))
}
