package cache

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0xfurai/gonest"
)

func TestCacheInterceptor_CachesGetRequests(t *testing.T) {
	store := NewMemoryStore()
	interceptor := NewCacheInterceptor(store, 5*time.Second)

	callCount := 0
	handler := func() (any, error) {
		callCount++
		return map[string]string{"data": "hello"}, nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/items?page=1", nil)
	ctx := makeExecCtx(w, r)

	// First call - should execute handler
	interceptor.Intercept(ctx, gonest.NewCallHandler(handler))
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// Second call - should hit cache
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/items?page=1", nil)
	ctx2 := makeExecCtx(w2, r2)
	interceptor.Intercept(ctx2, gonest.NewCallHandler(handler))
	if callCount != 1 {
		t.Errorf("expected still 1 call (cached), got %d", callCount)
	}
}

func TestCacheInterceptor_SkipsNonGet(t *testing.T) {
	store := NewMemoryStore()
	interceptor := NewCacheInterceptor(store, 5*time.Second)

	callCount := 0
	handler := func() (any, error) {
		callCount++
		return nil, nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/items", nil)
	ctx := makeExecCtx(w, r)

	interceptor.Intercept(ctx, gonest.NewCallHandler(handler))
	interceptor.Intercept(ctx, gonest.NewCallHandler(handler))
	if callCount != 2 {
		t.Errorf("expected 2 calls (no caching for POST), got %d", callCount)
	}
}

func TestCacheInterceptor_DifferentURLs(t *testing.T) {
	store := NewMemoryStore()
	interceptor := NewCacheInterceptor(store, 5*time.Second)

	callCount := 0
	handler := func() (any, error) {
		callCount++
		return "ok", nil
	}

	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/items?page=1", nil)
	interceptor.Intercept(makeExecCtx(w1, r1), gonest.NewCallHandler(handler))

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/items?page=2", nil)
	interceptor.Intercept(makeExecCtx(w2, r2), gonest.NewCallHandler(handler))

	if callCount != 2 {
		t.Errorf("expected 2 calls for different URLs, got %d", callCount)
	}
}

func TestCacheInterceptor_TTLExpiry(t *testing.T) {
	store := NewMemoryStore()
	interceptor := NewCacheInterceptor(store, 50*time.Millisecond)

	callCount := 0
	handler := func() (any, error) {
		callCount++
		return "ok", nil
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/items", nil)
	interceptor.Intercept(makeExecCtx(w, r), gonest.NewCallHandler(handler))

	time.Sleep(100 * time.Millisecond)

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/items", nil)
	interceptor.Intercept(makeExecCtx(w2, r2), gonest.NewCallHandler(handler))

	if callCount != 2 {
		t.Errorf("expected 2 calls after TTL expiry, got %d", callCount)
	}
}

func TestCacheInterceptor_HandlerError(t *testing.T) {
	store := NewMemoryStore()
	interceptor := NewCacheInterceptor(store, 5*time.Second)

	handler := func() (any, error) {
		return nil, gonest.NewInternalServerError("db error")
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/items", nil)
	_, err := interceptor.Intercept(makeExecCtx(w, r), gonest.NewCallHandler(handler))
	if err == nil {
		t.Fatal("expected error to propagate")
	}

	// Error responses should NOT be cached
	_, ok := store.Get("/items?")
	if ok {
		t.Error("error response should not be cached")
	}
}

// helper to build a minimal ExecutionContext for tests
type testExecCtx struct {
	gonest.Context
	req *http.Request
}

func (c *testExecCtx) GetHandler() any                         { return nil }
func (c *testExecCtx) GetClass() any                           { return nil }
func (c *testExecCtx) GetMetadata(key string) (any, bool)      { return nil, false }
func (c *testExecCtx) GetType() string                         { return "http" }
func (c *testExecCtx) SwitchToHTTP() gonest.HTTPContext         { return nil }
func (c *testExecCtx) Method() string                          { return c.req.Method }
func (c *testExecCtx) Path() string                            { return c.req.URL.Path }
func (c *testExecCtx) Request() *http.Request                  { return c.req }

func makeExecCtx(w http.ResponseWriter, r *http.Request) gonest.ExecutionContext {
	return &testExecCtx{req: r}
}
