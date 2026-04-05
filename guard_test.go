package gonest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGuardFunc_Allow(t *testing.T) {
	g := GuardFunc(func(ctx ExecutionContext) (bool, error) {
		return true, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, nil)

	allowed, err := g.CanActivate(execCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected allowed")
	}
}

func TestGuardFunc_Deny(t *testing.T) {
	g := GuardFunc(func(ctx ExecutionContext) (bool, error) {
		return false, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, nil)

	allowed, err := g.CanActivate(execCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected denied")
	}
}

func TestGuardFunc_Error(t *testing.T) {
	g := GuardFunc(func(ctx ExecutionContext) (bool, error) {
		return false, NewForbiddenException("not allowed")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, nil)

	_, err := g.CanActivate(execCtx)
	if err == nil {
		t.Fatal("expected error")
	}
	httpErr, ok := err.(*HTTPException)
	if !ok || httpErr.StatusCode() != http.StatusForbidden {
		t.Errorf("expected 403, got %v", err)
	}
}

func TestGuard_WithMetadata(t *testing.T) {
	g := GuardFunc(func(ctx ExecutionContext) (bool, error) {
		roles, ok := GetMetadata[[]string](ctx, "roles")
		if !ok {
			return true, nil
		}
		for _, r := range roles {
			if r == "admin" {
				return true, nil
			}
		}
		return false, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)

	// With admin role
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{
		"roles": []string{"admin"},
	})
	allowed, _ := g.CanActivate(execCtx)
	if !allowed {
		t.Error("expected admin to be allowed")
	}

	// Without admin role
	execCtx = newExecutionContext(ctx, nil, nil, map[string]any{
		"roles": []string{"user"},
	})
	allowed, _ = g.CanActivate(execCtx)
	if allowed {
		t.Error("expected non-admin to be denied")
	}

	// No roles metadata
	execCtx = newExecutionContext(ctx, nil, nil, map[string]any{})
	allowed, _ = g.CanActivate(execCtx)
	if !allowed {
		t.Error("expected access when no roles required")
	}
}
