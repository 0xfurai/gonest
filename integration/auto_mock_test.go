package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xfurai/gonest"
	nesttest "github.com/0xfurai/gonest/testing"
)

// ---------------------------------------------------------------------------
// Auto-Mock Integration Tests
// Mirror: original/integration/auto-mock/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Real services for auto-mock tests
// ---------------------------------------------------------------------------

type fooService struct {
	value string
}

func newFooService() *fooService {
	return &fooService{value: "real-foo"}
}

func (s *fooService) Foo() string {
	if s == nil {
		return ""
	}
	return s.value
}

type barService struct {
	foo *fooService
}

func newBarService(foo *fooService) *barService {
	return &barService{foo: foo}
}

func (s *barService) Bar() string {
	if s.foo == nil {
		return "bar-with-nil-foo"
	}
	return "bar-" + s.foo.Foo()
}

// ---------------------------------------------------------------------------
// Mock services
// ---------------------------------------------------------------------------

type mockFooService struct{}

func (s *mockFooService) Foo() string { return "mocked-foo" }

// ---------------------------------------------------------------------------
// Controller using barService
// ---------------------------------------------------------------------------

type barController struct {
	bar *barService
}

func newBarController(bar *barService) *barController {
	return &barController{bar: bar}
}

func (c *barController) Register(r gonest.Router) {
	r.Get("/bar", c.handler)
}

func (c *barController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"result": c.bar.Bar()})
}

func barModule() *gonest.Module {
	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newBarController},
		Providers:   []any{newFooService, newBarService},
	})
}

// ---------------------------------------------------------------------------
// Tests: MockFactory creates a mock provider
// ---------------------------------------------------------------------------

func TestAutoMock_MockFactory_CreatesMock(t *testing.T) {
	mockFoo := &fooService{value: "mock-value"}
	mock := nesttest.MockFactory[*fooService](func() *fooService {
		return mockFoo
	})

	compiled := nesttest.Test(barModule()).
		OverrideProvider((*fooService)(nil), mock).
		Compile(t)

	svc := nesttest.Resolve[*barService](compiled)
	result := svc.Bar()
	if result != "bar-mock-value" {
		t.Errorf("expected bar-mock-value, got %q", result)
	}
}

func TestAutoMock_MockFactory_WithController(t *testing.T) {
	mock := nesttest.MockFactory[*fooService](func() *fooService {
		return &fooService{value: "controller-mock"}
	})

	compiled := nesttest.Test(barModule()).
		OverrideProvider((*fooService)(nil), mock).
		Compile(t)

	app := compiled.App()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bar", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["result"] != "bar-controller-mock" {
		t.Errorf("expected bar-controller-mock, got %q", body["result"])
	}
}

// ---------------------------------------------------------------------------
// Tests: AutoMock creates a zero-value provider
// ---------------------------------------------------------------------------

func TestAutoMock_AutoMock_ZeroValue(t *testing.T) {
	mock := nesttest.AutoMock[*fooService]()

	compiled := nesttest.Test(barModule()).
		OverrideProvider((*fooService)(nil), mock).
		Compile(t)

	svc := nesttest.Resolve[*barService](compiled)
	// AutoMock returns nil for pointer types
	result := svc.Bar()
	if result != "bar-with-nil-foo" {
		t.Errorf("expected bar-with-nil-foo, got %q", result)
	}
}

func TestAutoMock_AutoMock_WithHTTP(t *testing.T) {
	mock := nesttest.AutoMock[*fooService]()

	compiled := nesttest.Test(barModule()).
		OverrideProvider((*fooService)(nil), mock).
		Compile(t)

	app := compiled.App()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/bar", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["result"] != "bar-with-nil-foo" {
		t.Errorf("expected bar-with-nil-foo, got %q", body["result"])
	}
}

// ---------------------------------------------------------------------------
// Tests: MockFactory with multiple overrides
// ---------------------------------------------------------------------------

func TestAutoMock_MultipleOverrides(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newFooService, newBarService},
	})

	mockFoo := nesttest.MockFactory[*fooService](func() *fooService {
		return &fooService{value: "override-1"}
	})
	mockBar := nesttest.MockFactory[*barService](func() *barService {
		return &barService{foo: &fooService{value: "override-2"}}
	})

	compiled := nesttest.Test(module).
		OverrideProvider((*fooService)(nil), mockFoo).
		OverrideProvider((*barService)(nil), mockBar).
		Compile(t)

	bar := nesttest.Resolve[*barService](compiled)
	result := bar.Bar()
	if result != "bar-override-2" {
		t.Errorf("expected bar-override-2, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// Tests: MockFactory with imported module
// ---------------------------------------------------------------------------

func TestAutoMock_MockFactory_WithOverrideModule(t *testing.T) {
	fooModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newFooService},
		Exports:   []any{(*fooService)(nil)},
	})

	fakeFooModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{nesttest.MockFactory[*fooService](func() *fooService {
			return &fooService{value: "imported-mock"}
		})},
		Exports: []any{(*fooService)(nil)},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{fooModule},
		Providers: []any{newBarService},
	})

	compiled := nesttest.Test(appModule).
		OverrideModule(fooModule, fakeFooModule).
		Compile(t)

	bar := nesttest.Resolve[*barService](compiled)
	result := bar.Bar()
	if result != "bar-imported-mock" {
		t.Errorf("expected bar-imported-mock, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// Tests: MockFactory preserves test module resolution
// ---------------------------------------------------------------------------

func TestAutoMock_ResolveOriginal_WhenNoOverride(t *testing.T) {
	compiled := nesttest.Test(barModule()).Compile(t)

	foo := nesttest.Resolve[*fooService](compiled)
	if foo.Foo() != "real-foo" {
		t.Errorf("expected real-foo without override, got %q", foo.Foo())
	}

	bar := nesttest.Resolve[*barService](compiled)
	if bar.Bar() != "bar-real-foo" {
		t.Errorf("expected bar-real-foo without override, got %q", bar.Bar())
	}
}
