package gonest

import (
	"net/http/httptest"
	"testing"
)

func TestMiddlewareFunc_Passthrough(t *testing.T) {
	called := false
	mw := MiddlewareFunc(func(ctx Context, next NextFunc) error {
		called = true
		return next()
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	nextCalled := false
	err := mw.Use(ctx, func() error {
		nextCalled = true
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected middleware to be called")
	}
	if !nextCalled {
		t.Error("expected next to be called")
	}
}

func TestMiddlewareFunc_ShortCircuit(t *testing.T) {
	mw := MiddlewareFunc(func(ctx Context, next NextFunc) error {
		return ctx.JSON(401, map[string]string{"error": "unauthorized"})
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	nextCalled := false
	mw.Use(ctx, func() error {
		nextCalled = true
		return nil
	})

	if nextCalled {
		t.Error("expected next NOT to be called")
	}
	if w.Code != 401 {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddlewareFunc_SetHeader(t *testing.T) {
	mw := MiddlewareFunc(func(ctx Context, next NextFunc) error {
		ctx.SetHeader("X-Request-ID", "123")
		return next()
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	mw.Use(ctx, func() error { return nil })

	if w.Header().Get("X-Request-ID") != "123" {
		t.Error("expected X-Request-ID header")
	}
}

func TestMiddleware_Chain(t *testing.T) {
	var order []int

	mw1 := MiddlewareFunc(func(ctx Context, next NextFunc) error {
		order = append(order, 1)
		err := next()
		order = append(order, 4)
		return err
	})

	mw2 := MiddlewareFunc(func(ctx Context, next NextFunc) error {
		order = append(order, 2)
		err := next()
		order = append(order, 3)
		return err
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	// Simulate chain
	mw1.Use(ctx, func() error {
		return mw2.Use(ctx, func() error {
			return nil
		})
	})

	expected := []int{1, 2, 3, 4}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("position %d: expected %d, got %d", i, expected[i], order[i])
		}
	}
}
