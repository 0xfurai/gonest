package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0xfurai/gonest"
)

// ---------------------------------------------------------------------------
// Nest Application Integration Tests
// Mirror: original/integration/nest-application/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// App controller for testing
// ---------------------------------------------------------------------------

type appController struct{}

func newAppController() *appController { return &appController{} }

func (c *appController) Register(r gonest.Router) {
	r.Get("/", c.root)
	r.Get("/test", c.test)
	r.Get("/middleware-test", c.mwTest)
	r.Post("/body", c.body)
}

func (c *appController) root(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": "root"})
}

func (c *appController) test(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": "test"})
}

func (c *appController) mwTest(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": "mw-test"})
}

func (c *appController) body(ctx gonest.Context) error {
	var b map[string]any
	if err := ctx.Bind(&b); err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, b)
}

// ---------------------------------------------------------------------------
// Tests: Global Prefix
// ---------------------------------------------------------------------------

func TestNestApplication_GlobalPrefix(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.SetGlobalPrefix("/api/v1")
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Route with prefix should work
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "test" {
		t.Errorf("expected test, got %q", body["message"])
	}

	// Route without prefix should 404
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/test", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestNestApplication_GlobalPrefix_Root(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.SetGlobalPrefix("/api")
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/, got %d", w.Code)
	}
}

func TestNestApplication_GlobalPrefix_AllRoutesPrefixed(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.SetGlobalPrefix("/api")
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	routes := app.GetRoutes()
	for _, route := range routes {
		// Route paths in GetRoutes don't include prefix, but when hit via HTTP they need prefix
		w := httptest.NewRecorder()
		path := "/api" + route.Path
		req := httptest.NewRequest(route.Method, path, nil)
		app.Handler().ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("route %s %s should be reachable", route.Method, path)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Handler access
// ---------------------------------------------------------------------------

func TestNestApplication_Handler(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	handler := app.Handler()
	if handler == nil {
		t.Fatal("Handler() returned nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: GetContainer
// ---------------------------------------------------------------------------

func TestNestApplication_GetContainer(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()
	if container == nil {
		t.Fatal("GetContainer() returned nil")
	}

	logger, err := gonest.Resolve[*loggerService](container)
	if err != nil {
		t.Fatal(err)
	}
	if logger == nil {
		t.Error("expected logger from container")
	}
}

// ---------------------------------------------------------------------------
// Tests: GetRoutes
// ---------------------------------------------------------------------------

func TestNestApplication_GetRoutes(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	routes := app.GetRoutes()
	if len(routes) == 0 {
		t.Fatal("expected registered routes")
	}

	// Should have at least /, /test, /middleware-test, /body
	routeMap := make(map[string]bool)
	for _, r := range routes {
		routeMap[r.Method+" "+r.Path] = true
	}

	expected := []string{"GET /", "GET /test", "GET /middleware-test", "POST /body"}
	for _, e := range expected {
		if !routeMap[e] {
			t.Errorf("missing route: %s", e)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Body Parsing
// ---------------------------------------------------------------------------

func TestNestApplication_BodyParsing_JSON(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	payload := `{"key":"value","number":42}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/body", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["key"] != "value" {
		t.Errorf("expected key=value, got %v", body["key"])
	}
	if body["number"] != float64(42) {
		t.Errorf("expected number=42, got %v", body["number"])
	}
}

func TestNestApplication_BodyParsing_EmptyBody(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/body", nil)
	app.Handler().ServeHTTP(w, req)

	// Should return 400 for empty body
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: SSE
// ---------------------------------------------------------------------------

func TestNestApplication_SSE_Stream(t *testing.T) {
	sseCtrl := &sseController{}
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *sseController { return sseCtrl }},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sse", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected text/event-stream, got %q", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control: no-cache")
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: message") {
		t.Errorf("expected SSE event in body, got %q", body)
	}
	if !strings.Contains(body, `"hello"`) {
		t.Errorf("expected hello data in body")
	}
}

type sseController struct{}

func (c *sseController) Register(r gonest.Router) {
	r.Get("/sse", gonest.SSE(func(stream *gonest.SSEStream) {
		stream.Send(gonest.SSEEvent{Event: "message", Data: "hello"})
		stream.Send(gonest.SSEEvent{Event: "message", Data: "world"})
		stream.Close()
	}))
}

// ---------------------------------------------------------------------------
// Tests: Multiple Modules with Global Prefix
// ---------------------------------------------------------------------------

func TestNestApplication_GlobalPrefix_MultiModule(t *testing.T) {
	catMod := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newCatController},
		Providers:   []any{newCatService},
	})
	rootMod := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{catMod},
		Controllers: []any{newAppController},
	})

	app := gonest.Create(rootMod, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.SetGlobalPrefix("/api")
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Root module route
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/test, got %d", w.Code)
	}

	// Imported module route
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/cats", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 on /api/cats, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Close is idempotent
// ---------------------------------------------------------------------------

func TestNestApplication_Close_Idempotent(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}

	if err := app.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	// Second close should not panic
	app.Close()
}

// ---------------------------------------------------------------------------
// Tests: Multiple Global Features
// ---------------------------------------------------------------------------

func TestNestApplication_CombinedGlobalFeatures(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.SetGlobalPrefix("/api")
	app.EnableCors(gonest.CorsOptions{Origin: "http://app.test"})
	app.UseGlobalMiddleware(gonest.MiddlewareFunc(func(ctx gonest.Context, next gonest.NextFunc) error {
		ctx.SetHeader("X-Request-ID", "test-123")
		return next()
	}))

	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://app.test")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "http://app.test" {
		t.Error("CORS not applied")
	}
	if w.Header().Get("X-Request-ID") != "test-123" {
		t.Error("middleware not applied")
	}
}

// ---------------------------------------------------------------------------
// Tests: Listen on real port then Close
// Mirror: original/integration/nest-application/e2e/listen
// ---------------------------------------------------------------------------

func TestNestApplication_ListenAndClose(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newAppController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})

	// Get a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	addr := fmt.Sprintf(":%d", port)

	go func() {
		app.Listen(addr)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Make a real HTTP request
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/test", port))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["message"] != "test" {
		t.Errorf("expected test, got %q", body["message"])
	}

	// Close the server
	app.Close()
}

// ---------------------------------------------------------------------------
// Tests: Raw body access
// Mirror: original/integration/nest-application/e2e/raw-body
// ---------------------------------------------------------------------------

type rawBodyController struct{}

func newRawBodyController() *rawBodyController { return &rawBodyController{} }

func (c *rawBodyController) Register(r gonest.Router) {
	r.Post("/raw", c.handler)
}

func (c *rawBodyController) handler(ctx gonest.Context) error {
	data, err := io.ReadAll(ctx.Body())
	if err != nil {
		return gonest.NewBadRequestException("failed to read body")
	}
	return ctx.String(http.StatusOK, string(data))
}

func TestNestApplication_RawBodyAccess(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newRawBodyController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	payload := "raw body content"
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/raw", strings.NewReader(payload))
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != payload {
		t.Errorf("expected %q, got %q", payload, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tests: Query parameters
// Mirror: original/integration/nest-application context usage
// ---------------------------------------------------------------------------

type queryController struct{}

func newQueryController() *queryController { return &queryController{} }

func (c *queryController) Register(r gonest.Router) {
	r.Get("/search", c.search)
}

func (c *queryController) search(ctx gonest.Context) error {
	q := ctx.Query("q")
	page := ctx.Query("page")
	return ctx.JSON(http.StatusOK, map[string]string{"q": q, "page": page})
}

func TestNestApplication_QueryParams(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newQueryController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/search?q=test&page=2", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["q"] != "test" {
		t.Errorf("expected q=test, got %q", body["q"])
	}
	if body["page"] != "2" {
		t.Errorf("expected page=2, got %q", body["page"])
	}
}

// ---------------------------------------------------------------------------
// Tests: Cookie setting/reading
// ---------------------------------------------------------------------------

type cookieController struct{}

func newCookieController() *cookieController { return &cookieController{} }

func (c *cookieController) Register(r gonest.Router) {
	r.Get("/set-cookie", c.setCookie)
	r.Get("/get-cookie", c.getCookie)
}

func (c *cookieController) setCookie(ctx gonest.Context) error {
	ctx.SetCookie(&http.Cookie{Name: "session", Value: "abc123", Path: "/"})
	return ctx.String(http.StatusOK, "cookie set")
}

func (c *cookieController) getCookie(ctx gonest.Context) error {
	cookie, err := ctx.Cookie("session")
	if err != nil {
		return ctx.String(http.StatusOK, "no cookie")
	}
	return ctx.String(http.StatusOK, cookie.Value)
}

func TestNestApplication_Cookies(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newCookieController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Set cookie
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/set-cookie", nil)
	app.Handler().ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header")
	}
	if cookies[0].Name != "session" || cookies[0].Value != "abc123" {
		t.Errorf("expected session=abc123, got %s=%s", cookies[0].Name, cookies[0].Value)
	}

	// Read cookie back
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/get-cookie", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	app.Handler().ServeHTTP(w, req)

	if w.Body.String() != "abc123" {
		t.Errorf("expected abc123, got %q", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tests: IP extraction
// ---------------------------------------------------------------------------

func TestNestApplication_IPExtraction(t *testing.T) {
	ctrl := &ipController{}
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *ipController { return ctrl }},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// With X-Forwarded-For
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ip", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	app.Handler().ServeHTTP(w, req)

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["ip"] != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %q", body["ip"])
	}
}

type ipController struct{}

func (c *ipController) Register(r gonest.Router) {
	r.Get("/ip", c.handler)
}

func (c *ipController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"ip": ctx.IP()})
}
