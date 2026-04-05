package gonest

import (
	"net/http/httptest"
	"testing"
)

func TestVersioningMiddleware_URI(t *testing.T) {
	mw := NewVersioningMiddleware(VersioningOptions{Type: VersioningURI})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/v2/cats", nil)
	ctx := newContext(w, r)

	mw.Use(ctx, func() error { return nil })

	v := GetVersion(ctx)
	if v != "2" {
		t.Errorf("expected '2', got %q", v)
	}
}

func TestVersioningMiddleware_Header(t *testing.T) {
	mw := NewVersioningMiddleware(VersioningOptions{Type: VersioningHeader})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	r.Header.Set("X-API-Version", "3")
	ctx := newContext(w, r)

	mw.Use(ctx, func() error { return nil })

	v := GetVersion(ctx)
	if v != "3" {
		t.Errorf("expected '3', got %q", v)
	}
}

func TestVersioningMiddleware_CustomHeader(t *testing.T) {
	mw := NewVersioningMiddleware(VersioningOptions{
		Type:   VersioningHeader,
		Header: "Api-Version",
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	r.Header.Set("Api-Version", "5")
	ctx := newContext(w, r)

	mw.Use(ctx, func() error { return nil })
	if GetVersion(ctx) != "5" {
		t.Errorf("expected '5', got %q", GetVersion(ctx))
	}
}

func TestVersioningMiddleware_MediaType(t *testing.T) {
	mw := NewVersioningMiddleware(VersioningOptions{Type: VersioningMediaType})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	r.Header.Set("Accept", "application/json;v=2")
	ctx := newContext(w, r)

	mw.Use(ctx, func() error { return nil })
	if GetVersion(ctx) != "2" {
		t.Errorf("expected '2', got %q", GetVersion(ctx))
	}
}

func TestVersioningMiddleware_Default(t *testing.T) {
	mw := NewVersioningMiddleware(VersioningOptions{
		Type:           VersioningHeader,
		DefaultVersion: "1",
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	ctx := newContext(w, r)

	mw.Use(ctx, func() error { return nil })
	if GetVersion(ctx) != "1" {
		t.Errorf("expected default '1', got %q", GetVersion(ctx))
	}
}

func TestVersionGuard_MatchingVersion(t *testing.T) {
	guard := NewVersionGuard()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	ctx := newContext(w, r)
	ctx.Set("__api_version", "2")
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{"version": "2"})

	allowed, err := guard.CanActivate(execCtx)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("expected allowed for matching version")
	}
}

func TestVersionGuard_MismatchVersion(t *testing.T) {
	guard := NewVersionGuard()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	ctx := newContext(w, r)
	ctx.Set("__api_version", "1")
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{"version": "2"})

	_, err := guard.CanActivate(execCtx)
	if err == nil {
		t.Fatal("expected error for version mismatch")
	}
}

func TestVersionGuard_NoVersionRequired(t *testing.T) {
	guard := NewVersionGuard()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{})

	allowed, _ := guard.CanActivate(execCtx)
	if !allowed {
		t.Error("expected allowed when no version required")
	}
}

func TestExtractURIVersion(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/v1/cats", "1"},
		{"/v2/users/42", "2"},
		{"/cats", ""},
		{"/api/cats", ""},
	}
	for _, tt := range tests {
		result := extractURIVersion(tt.path)
		if result != tt.expected {
			t.Errorf("extractURIVersion(%q): expected %q, got %q", tt.path, tt.expected, result)
		}
	}
}

func TestVersioningMiddleware_Custom(t *testing.T) {
	mw := NewVersioningMiddleware(VersioningOptions{
		Type: VersioningCustom,
		Extractor: func(ctx Context) string {
			return ctx.Query("api-version")
		},
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats?api-version=4", nil)
	ctx := newContext(w, r)

	mw.Use(ctx, func() error { return nil })
	if GetVersion(ctx) != "4" {
		t.Errorf("expected '4', got %q", GetVersion(ctx))
	}
}

func TestVersionGuard_VersionNeutral(t *testing.T) {
	guard := NewVersionGuard()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/cats", nil)
	ctx := newContext(w, r)
	ctx.Set("__api_version", "99")
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{"version": VersionNeutral})

	allowed, err := guard.CanActivate(execCtx)
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("expected allowed for version-neutral route")
	}
}

func TestExtractMediaTypeVersion(t *testing.T) {
	tests := []struct {
		accept   string
		expected string
	}{
		{"application/json;v=2", "2"},
		{"application/json", ""},
		{"text/html;v=1", "1"},
	}
	for _, tt := range tests {
		result := extractMediaTypeVersion(tt.accept)
		if result != tt.expected {
			t.Errorf("extractMediaTypeVersion(%q): expected %q, got %q", tt.accept, tt.expected, result)
		}
	}
}
