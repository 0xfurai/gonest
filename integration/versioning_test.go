package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonest"
)

// ---------------------------------------------------------------------------
// Versioning Integration Tests
// Mirror: original/integration/versioning/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Versioned controllers
// ---------------------------------------------------------------------------

type versionedController struct{}

func newVersionedController() *versionedController { return &versionedController{} }

func (c *versionedController) Register(r gonest.Router) {
	r.Get("/v1/cats", c.v1Cats).SetMetadata("version", "1")
	r.Get("/v2/cats", c.v2Cats).SetMetadata("version", "2")
	r.Get("/neutral", c.neutral)
}

func (c *versionedController) v1Cats(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"version": 1, "cats": []string{"Whiskers"}})
}

func (c *versionedController) v2Cats(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"version": 2, "cats": []string{"Whiskers", "Tom"}})
}

func (c *versionedController) neutral(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": "neutral"})
}

// Header-based versioned controller
type headerVersionController struct{}

func newHeaderVersionController() *headerVersionController { return &headerVersionController{} }

func (c *headerVersionController) Register(r gonest.Router) {
	r.Prefix("/api")
	r.Get("/data", c.data).SetMetadata("version", "1")
	r.Get("/data-v2", c.dataV2).SetMetadata("version", "2")
}

func (c *headerVersionController) data(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"version": 1})
}

func (c *headerVersionController) dataV2(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"version": 2})
}

// ---------------------------------------------------------------------------
// Tests: URI Versioning
// ---------------------------------------------------------------------------

func TestVersioning_URI_V1Route(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newVersionedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningURI,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/cats", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["version"] != float64(1) {
		t.Errorf("expected version 1, got %v", body["version"])
	}
}

func TestVersioning_URI_V2Route(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newVersionedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningURI,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v2/cats", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["version"] != float64(2) {
		t.Errorf("expected version 2, got %v", body["version"])
	}
	cats := body["cats"].([]any)
	if len(cats) != 2 {
		t.Errorf("expected 2 cats in v2, got %d", len(cats))
	}
}

func TestVersioning_URI_NeutralRoute(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newVersionedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningURI,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/neutral", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for neutral route, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Header Versioning
// ---------------------------------------------------------------------------

func TestVersioning_Header_MatchesVersion(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningHeader,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Correct version header
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Version", "1")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestVersioning_Header_WrongVersion(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningHeader,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Wrong version header for v1 route
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-API-Version", "2")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for wrong version, got %d", w.Code)
	}
}

func TestVersioning_Header_NoVersionAllowsAccess(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningHeader,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// No version header - VersionGuard allows when no version in request
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when no version header, got %d", w.Code)
	}
}

func TestVersioning_Header_CustomHeaderName(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type:   gonest.VersioningHeader,
		Header: "X-Custom-Version",
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-Custom-Version", "1")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Media Type Versioning
// ---------------------------------------------------------------------------

func TestVersioning_MediaType_MatchesVersion(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningMediaType,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Accept", "application/json;v=1")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestVersioning_MediaType_WrongVersion(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningMediaType,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Accept", "application/json;v=2")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Default Version
// ---------------------------------------------------------------------------

func TestVersioning_DefaultVersion(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type:           gonest.VersioningHeader,
		DefaultVersion: "1",
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// No version header provided — should default to "1"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with default version, got %d", w.Code)
	}
}

func TestVersioning_DefaultVersion_Mismatch(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderVersionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type:           gonest.VersioningHeader,
		DefaultVersion: "99",
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// No version header, default is "99" — route expects "1"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/data", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 with mismatched default version, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Versioning with Global Prefix
// ---------------------------------------------------------------------------

func TestVersioning_WithGlobalPrefix(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newVersionedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.SetGlobalPrefix("/api")
	app.UseGlobalMiddleware(gonest.NewVersioningMiddleware(gonest.VersioningOptions{
		Type: gonest.VersioningURI,
	}))
	app.UseGlobalGuards(gonest.NewVersionGuard())
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/cats", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
