package gonest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInterceptorFunc_Passthrough(t *testing.T) {
	interceptor := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		return next.Handle()
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, nil)

	called := false
	next := NewCallHandler(func() (any, error) {
		called = true
		return "result", nil
	})

	result, err := interceptor.Intercept(execCtx, next)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected next to be called")
	}
	if result != "result" {
		t.Errorf("expected 'result', got %v", result)
	}
}

func TestInterceptor_ModifyResponse(t *testing.T) {
	interceptor := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		ctx.SetHeader("X-Modified", "true")
		result, err := next.Handle()
		return result, err
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, nil)

	next := NewCallHandler(func() (any, error) {
		return nil, nil
	})

	interceptor.Intercept(execCtx, next)
	if w.Header().Get("X-Modified") != "true" {
		t.Error("expected modified header")
	}
}

func TestInterceptor_ShortCircuit(t *testing.T) {
	interceptor := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		return nil, NewForbiddenException("blocked by interceptor")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, nil)

	nextCalled := false
	next := NewCallHandler(func() (any, error) {
		nextCalled = true
		return nil, nil
	})

	_, err := interceptor.Intercept(execCtx, next)
	if err == nil {
		t.Fatal("expected error")
	}
	if nextCalled {
		t.Error("expected next NOT to be called")
	}
	httpErr := err.(*HTTPException)
	if httpErr.StatusCode() != http.StatusForbidden {
		t.Errorf("expected 403, got %d", httpErr.StatusCode())
	}
}

func TestInterceptor_TransformResult(t *testing.T) {
	interceptor := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		result, err := next.Handle()
		if err != nil {
			return nil, err
		}
		// Wrap the result
		return map[string]any{"data": result}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, nil)

	next := NewCallHandler(func() (any, error) {
		return "original", nil
	})

	result, err := interceptor.Intercept(execCtx, next)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wrapped := result.(map[string]any)
	if wrapped["data"] != "original" {
		t.Errorf("expected wrapped result, got %v", result)
	}
}
