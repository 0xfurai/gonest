package gonest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestThrottleGuard_AllowsWithinLimit(t *testing.T) {
	guard := NewThrottleGuard(3, time.Second)

	for i := 0; i < 3; i++ {
		ctx := makeThrottleCtx("192.168.1.1")
		allowed, err := guard.CanActivate(ctx)
		if err != nil {
			t.Fatalf("request %d: unexpected error: %v", i, err)
		}
		if !allowed {
			t.Errorf("request %d: expected allowed", i)
		}
	}
}

func TestThrottleGuard_BlocksOverLimit(t *testing.T) {
	guard := NewThrottleGuard(2, time.Second)

	makeThrottleCtx("10.0.0.1")
	guard.CanActivate(makeThrottleCtx("10.0.0.1"))
	guard.CanActivate(makeThrottleCtx("10.0.0.1"))

	allowed, err := guard.CanActivate(makeThrottleCtx("10.0.0.1"))
	if err == nil {
		t.Fatal("expected error for rate limit exceeded")
	}
	if allowed {
		t.Error("expected blocked")
	}
	httpErr, ok := err.(*HTTPException)
	if !ok || httpErr.StatusCode() != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %v", err)
	}
}

func TestThrottleGuard_ResetsAfterWindow(t *testing.T) {
	guard := NewThrottleGuard(1, 50*time.Millisecond)

	guard.CanActivate(makeThrottleCtx("10.0.0.1"))
	// Should be blocked
	_, err := guard.CanActivate(makeThrottleCtx("10.0.0.1"))
	if err == nil {
		t.Fatal("expected rate limit error")
	}

	// Wait for window reset
	time.Sleep(60 * time.Millisecond)

	allowed, err := guard.CanActivate(makeThrottleCtx("10.0.0.1"))
	if err != nil {
		t.Fatalf("expected allowed after reset: %v", err)
	}
	if !allowed {
		t.Error("expected allowed after window reset")
	}
}

func TestThrottleGuard_DifferentIPs(t *testing.T) {
	guard := NewThrottleGuard(1, time.Second)

	guard.CanActivate(makeThrottleCtx("1.1.1.1"))
	// Different IP should still be allowed
	allowed, err := guard.CanActivate(makeThrottleCtx("2.2.2.2"))
	if err != nil {
		t.Fatalf("different IP should be allowed: %v", err)
	}
	if !allowed {
		t.Error("different IP should be allowed")
	}
}

func TestThrottleByMetadataGuard_CustomLimits(t *testing.T) {
	guard := NewThrottleByMetadataGuard(100, time.Minute)

	// Route with custom limit of 1
	ctx := makeThrottleCtxWithMeta("10.0.0.1", "/api/expensive", map[string]any{
		"throttle_limit": 1,
	})
	guard.CanActivate(ctx)

	ctx2 := makeThrottleCtxWithMeta("10.0.0.1", "/api/expensive", map[string]any{
		"throttle_limit": 1,
	})
	_, err := guard.CanActivate(ctx2)
	if err == nil {
		t.Fatal("expected rate limit for custom limit=1")
	}
}

func makeThrottleCtx(ip string) ExecutionContext {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = ip + ":1234"
	ctx := newContext(w, r)
	return newExecutionContext(ctx, nil, nil, map[string]any{})
}

func makeThrottleCtxWithMeta(ip, path string, meta map[string]any) ExecutionContext {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", path, nil)
	r.RemoteAddr = ip + ":1234"
	ctx := newContext(w, r)
	return newExecutionContext(ctx, nil, nil, meta)
}
