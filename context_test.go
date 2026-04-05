package gonest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContext_JSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	err := ctx.JSON(http.StatusOK, map[string]string{"message": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
		t.Errorf("expected json content type, got %q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), `"message":"hello"`) {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestContext_String(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	err := ctx.String(http.StatusOK, "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Body.String() != "hello world" {
		t.Errorf("expected 'hello world', got %q", w.Body.String())
	}
}

func TestContext_Bind(t *testing.T) {
	body := `{"name":"Pixel","age":2}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	ctx := newContext(w, r)

	var dto struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	err := ctx.Bind(&dto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dto.Name != "Pixel" || dto.Age != 2 {
		t.Errorf("unexpected dto: %+v", dto)
	}
}

func TestContext_Bind_NilBody(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Body = nil
	ctx := newContext(w, r)

	var dto struct{}
	err := ctx.Bind(&dto)
	if err == nil {
		t.Fatal("expected error for nil body")
	}
}

func TestContext_Bind_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", strings.NewReader("not json"))
	ctx := newContext(w, r)

	var dto struct{}
	err := ctx.Bind(&dto)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestContext_Param(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users/42", nil)
	ctx := newContext(w, r)
	ctx.setParam("id", "42")

	if ctx.Param("id") != "42" {
		t.Errorf("expected '42', got %v", ctx.Param("id"))
	}
	if ctx.Param("missing") != nil {
		t.Errorf("expected nil for missing param")
	}
}

func TestContext_Query(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/search?q=cats&page=2", nil)
	ctx := newContext(w, r)

	if ctx.Query("q") != "cats" {
		t.Errorf("expected 'cats', got %q", ctx.Query("q"))
	}
	if ctx.Query("page") != "2" {
		t.Errorf("expected '2', got %q", ctx.Query("page"))
	}
}

func TestContext_Header(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer token")
	ctx := newContext(w, r)

	if ctx.Header("Authorization") != "Bearer token" {
		t.Errorf("unexpected header: %q", ctx.Header("Authorization"))
	}
}

func TestContext_SetAndGet(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	ctx.Set("user", "john")
	val, ok := ctx.Get("user")
	if !ok || val != "john" {
		t.Errorf("expected 'john', got %v", val)
	}

	_, ok = ctx.Get("missing")
	if ok {
		t.Error("expected not found for missing key")
	}
}

func TestContext_NoContent(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/", nil)
	ctx := newContext(w, r)

	ctx.NoContent(http.StatusNoContent)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestContext_Redirect(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/old", nil)
	ctx := newContext(w, r)

	ctx.Redirect(http.StatusFound, "/new")
	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}
}

func TestContext_IP(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.1:1234"
	ctx := newContext(w, r)

	if ctx.IP() != "192.168.1.1:1234" {
		t.Errorf("unexpected IP: %q", ctx.IP())
	}
}

func TestContext_IP_XForwardedFor(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1")
	ctx := newContext(w, r)

	if ctx.IP() != "10.0.0.1" {
		t.Errorf("expected '10.0.0.1', got %q", ctx.IP())
	}
}

func TestContext_MethodAndPath(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/cats", nil)
	ctx := newContext(w, r)

	if ctx.Method() != "POST" {
		t.Errorf("expected POST, got %q", ctx.Method())
	}
	if ctx.Path() != "/api/cats" {
		t.Errorf("expected '/api/cats', got %q", ctx.Path())
	}
}

func TestContext_SetHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	ctx.SetHeader("X-Custom", "value")
	if w.Header().Get("X-Custom") != "value" {
		t.Errorf("expected 'value', got %q", w.Header().Get("X-Custom"))
	}
}

func TestContext_Cookie(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	ctx := newContext(w, r)

	cookie, err := ctx.Cookie("session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cookie.Value != "abc123" {
		t.Errorf("expected 'abc123', got %q", cookie.Value)
	}
}

func TestContext_SetCookie(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	ctx.SetCookie(&http.Cookie{Name: "token", Value: "xyz"})
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "token" {
		t.Error("expected cookie to be set")
	}
}

func TestContext_QueryValues(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/search?a=1&b=2", nil)
	ctx := newContext(w, r)

	vals := ctx.QueryValues()
	if vals.Get("a") != "1" || vals.Get("b") != "2" {
		t.Errorf("unexpected query values: %v", vals)
	}
}

func TestContext_Written(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	if ctx.Written() {
		t.Error("expected Written() to be false before write")
	}
	ctx.JSON(200, nil)
	if !ctx.Written() {
		t.Error("expected Written() to be true after write")
	}
}

func TestContext_Status(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	result := ctx.Status(201)
	if result != ctx {
		t.Error("Status() should return the same context for chaining")
	}
}

func TestContext_JSON_NilValue(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	err := ctx.JSON(http.StatusNoContent, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", w.Body.String())
	}
}

func TestExecutionContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	execCtx := newExecutionContext(ctx, "handler1", "controller1", map[string]any{
		"roles": []string{"admin"},
	})

	if execCtx.GetHandler() != "handler1" {
		t.Errorf("unexpected handler: %v", execCtx.GetHandler())
	}
	if execCtx.GetClass() != "controller1" {
		t.Errorf("unexpected controller: %v", execCtx.GetClass())
	}
	if execCtx.GetType() != "http" {
		t.Errorf("expected 'http', got %q", execCtx.GetType())
	}

	roles, ok := execCtx.GetMetadata("roles")
	if !ok {
		t.Fatal("expected metadata")
	}
	if len(roles.([]string)) != 1 {
		t.Errorf("expected 1 role, got %v", roles)
	}
}

func TestHTTPContext(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	ctx := newContext(w, r)

	execCtx := newExecutionContext(ctx, nil, nil, nil)
	httpCtx := execCtx.SwitchToHTTP()

	if httpCtx.Request().URL.Path != "/test" {
		t.Errorf("expected '/test', got %q", httpCtx.Request().URL.Path)
	}
}

func TestArgumentsHost(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	ctx := newContext(w, r)

	host := newArgumentsHost(ctx)
	if host.GetType() != "http" {
		t.Errorf("expected 'http', got %q", host.GetType())
	}

	httpHost := host.SwitchToHTTP()
	if httpHost.Request().URL.Path != "/test" {
		t.Errorf("expected '/test', got %q", httpHost.Request().URL.Path)
	}
}

func TestCallHandler(t *testing.T) {
	ch := NewCallHandler(func() (any, error) {
		return "result", nil
	})

	result, err := ch.Handle()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}
}

func TestResponseWriter(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{w: w}

	rw.Status(201).JSON(map[string]string{"ok": "true"})
	if w.Code != 201 {
		t.Errorf("expected 201, got %d", w.Code)
	}
}
