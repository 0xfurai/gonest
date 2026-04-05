package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xfurai/gonest"
)

// ---------------------------------------------------------------------------
// CORS Integration Tests
// Mirror: original/integration/cors/
// ---------------------------------------------------------------------------

type corsController struct{}

func newCorsController() *corsController { return &corsController{} }

func (c *corsController) Register(r gonest.Router) {
	r.Get("/test", c.handler)
	r.Post("/test", c.handler)
}

func (c *corsController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func createCorsApp(t *testing.T, opts ...gonest.CorsOptions) *gonest.Application {
	t.Helper()
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newCorsController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.EnableCors(opts...)
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestCORS_DefaultWildcardOrigin(t *testing.T) {
	app := createCorsApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected ACAO=*, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_SpecificOrigin(t *testing.T) {
	app := createCorsApp(t, gonest.CorsOptions{
		Origin: "http://example.com",
	})
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected ACAO=http://example.com, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_OptionsPreflightReturns204(t *testing.T) {
	app := createCorsApp(t, gonest.CorsOptions{
		Origin: "http://example.com",
	})
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("expected ACAO header on preflight")
	}
}

func TestCORS_AllowMethods(t *testing.T) {
	app := createCorsApp(t, gonest.CorsOptions{
		Origin:  "*",
		Methods: "GET, POST",
	})
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" {
		t.Errorf("expected methods GET, POST, got %q", w.Header().Get("Access-Control-Allow-Methods"))
	}
}

func TestCORS_AllowHeaders(t *testing.T) {
	app := createCorsApp(t, gonest.CorsOptions{
		Origin:  "*",
		Headers: "Content-Type, X-Custom-Header",
	})
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type, X-Custom-Header" {
		t.Errorf("expected custom headers, got %q", w.Header().Get("Access-Control-Allow-Headers"))
	}
}

func TestCORS_Credentials(t *testing.T) {
	app := createCorsApp(t, gonest.CorsOptions{
		Origin:      "http://example.com",
		Credentials: true,
	})
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("expected ACAC=true, got %q", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORS_NoCredentialsByDefault(t *testing.T) {
	app := createCorsApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Errorf("expected no ACAC header, got %q", w.Header().Get("Access-Control-Allow-Credentials"))
	}
}

func TestCORS_PreflightDoesNotReachHandler(t *testing.T) {
	app := createCorsApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	// Body should be empty (handler not reached)
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body for preflight, got %q", w.Body.String())
	}
}

func TestCORS_DefaultAllowHeaders(t *testing.T) {
	app := createCorsApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	app.Handler().ServeHTTP(w, req)

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "Content-Type, Authorization" {
		t.Errorf("expected default headers, got %q", headers)
	}
}
