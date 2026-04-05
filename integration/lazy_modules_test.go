package integration

import (
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/0xfurai/gonest"
)

// ---------------------------------------------------------------------------
// Lazy Module Loading Integration Tests
// Mirror: original/integration/lazy-modules/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Services for lazy loading tests
// ---------------------------------------------------------------------------

type expensiveService struct {
	Name string
}

func newExpensiveService() *expensiveService {
	return &expensiveService{Name: "expensive"}
}

type lazyGlobalService struct {
	Value string
}

func newLazyGlobalService() *lazyGlobalService {
	return &lazyGlobalService{Value: "global-data"}
}

// ---------------------------------------------------------------------------
// Tests: Basic lazy module loading
// ---------------------------------------------------------------------------

func TestLazyModules_BasicLoad(t *testing.T) {
	rootModule := gonest.NewModule(gonest.ModuleOptions{})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	loader := app.GetLazyModuleLoader()

	lm, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{newExpensiveService},
			Exports:   []any{(*expensiveService)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	svc, err := lm.Get(reflect.TypeOf((*expensiveService)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	if svc == nil {
		t.Fatal("expected expensive service from lazy module")
	}

	typed := svc.(*expensiveService)
	if typed.Name != "expensive" {
		t.Errorf("expected name=expensive, got %q", typed.Name)
	}
}

func TestLazyModules_GenericResolve(t *testing.T) {
	rootModule := gonest.NewModule(gonest.ModuleOptions{})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	loader := app.GetLazyModuleLoader()

	lm, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{newExpensiveService},
			Exports:   []any{(*expensiveService)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	svc, err := gonest.LazyModuleResolve[*expensiveService](lm)
	if err != nil {
		t.Fatal(err)
	}
	if svc.Name != "expensive" {
		t.Errorf("expected name=expensive, got %q", svc.Name)
	}
}

// ---------------------------------------------------------------------------
// Tests: Lazy module with global module access
// ---------------------------------------------------------------------------

func TestLazyModules_AccessGlobalModule(t *testing.T) {
	type lazyConsumer struct {
		global *lazyGlobalService
	}

	globalModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLazyGlobalService},
		Exports:   []any{(*lazyGlobalService)(nil)},
		Global:    true,
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{globalModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	loader := app.GetLazyModuleLoader()

	lm, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{func(g *lazyGlobalService) *lazyConsumer {
				return &lazyConsumer{global: g}
			}},
			Exports: []any{(*lazyConsumer)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	consumer, err := lm.Get(reflect.TypeOf((*lazyConsumer)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	typed := consumer.(*lazyConsumer)
	if typed.global == nil {
		t.Fatal("expected global service to be injected into lazy module")
	}
	if typed.global.Value != "global-data" {
		t.Errorf("expected Value=global-data, got %q", typed.global.Value)
	}
}

// ---------------------------------------------------------------------------
// Tests: Lazy module lifecycle hooks
// ---------------------------------------------------------------------------

type lazyBootstrapService struct {
	bootstrapped bool
}

func newLazyBootstrapService() *lazyBootstrapService {
	return &lazyBootstrapService{}
}

func (s *lazyBootstrapService) OnApplicationBootstrap() error {
	s.bootstrapped = true
	return nil
}

func TestLazyModules_OnApplicationBootstrap_Called(t *testing.T) {
	rootModule := gonest.NewModule(gonest.ModuleOptions{})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	loader := app.GetLazyModuleLoader()

	lm, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{newLazyBootstrapService},
			Exports:   []any{(*lazyBootstrapService)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	svc, err := lm.Get(reflect.TypeOf((*lazyBootstrapService)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	typed := svc.(*lazyBootstrapService)
	if !typed.bootstrapped {
		t.Error("expected OnApplicationBootstrap to be called on lazy-loaded service")
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple lazy loads
// ---------------------------------------------------------------------------

func TestLazyModules_MultipleLazyLoads(t *testing.T) {
	rootModule := gonest.NewModule(gonest.ModuleOptions{})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	loader := app.GetLazyModuleLoader()

	// Load first module
	lm1, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{newExpensiveService},
			Exports:   []any{(*expensiveService)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	// Load second module with a different provider
	type secondService struct{ Value int }
	lm2, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{gonest.ProvideValue[*secondService](&secondService{Value: 42})},
			Exports:   []any{(*secondService)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	// Both should resolve independently
	svc1, err := lm1.Get(reflect.TypeOf((*expensiveService)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	if svc1 == nil {
		t.Error("expected expensive service from first lazy module")
	}

	svc2, err := lm2.Get(reflect.TypeOf((*secondService)(nil)))
	if err != nil {
		t.Fatal(err)
	}
	if svc2.(*secondService).Value != 42 {
		t.Error("expected Value=42 from second lazy module")
	}

	loaded := loader.GetLoadedModules()
	if len(loaded) != 2 {
		t.Errorf("expected 2 loaded modules, got %d", len(loaded))
	}
}

// ---------------------------------------------------------------------------
// Tests: Lazy module nil factory returns error
// ---------------------------------------------------------------------------

func TestLazyModules_NilFactory_ReturnsError(t *testing.T) {
	rootModule := gonest.NewModule(gonest.ModuleOptions{})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	loader := app.GetLazyModuleLoader()

	_, err := loader.Load(func() *gonest.Module {
		return nil
	})
	if err == nil {
		t.Fatal("expected error when factory returns nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: Lazy module updates discovery service
// ---------------------------------------------------------------------------

func TestLazyModules_UpdatesDiscoveryService(t *testing.T) {
	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newWebhookService},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	providersBefore := ds.GetProviders()
	countBefore := len(providersBefore)

	loader := app.GetLazyModuleLoader()
	_, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{newExpensiveService},
			Exports:   []any{(*expensiveService)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	providersAfter := ds.GetProviders()
	if len(providersAfter) <= countBefore {
		t.Error("expected discovery service to include providers from lazy-loaded module")
	}
}

// ---------------------------------------------------------------------------
// Tests: Lazy module with transient scope
// ---------------------------------------------------------------------------

var lazyTransientCounter atomic.Int64

type lazyTransientService struct {
	id int64
}

func newLazyTransientService() *lazyTransientService {
	return &lazyTransientService{id: lazyTransientCounter.Add(1)}
}

func TestLazyModules_TransientScope(t *testing.T) {
	lazyTransientCounter.Store(0)

	rootModule := gonest.NewModule(gonest.ModuleOptions{})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	loader := app.GetLazyModuleLoader()

	lm, err := loader.Load(func() *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{
				gonest.ProvideWithScope(newLazyTransientService, gonest.ScopeTransient),
			},
			Exports: []any{(*lazyTransientService)(nil)},
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	svcType := reflect.TypeOf((*lazyTransientService)(nil))
	svc1, err := lm.Get(svcType)
	if err != nil {
		t.Fatal(err)
	}
	svc2, err := lm.Get(svcType)
	if err != nil {
		t.Fatal(err)
	}

	s1 := svc1.(*lazyTransientService)
	s2 := svc2.(*lazyTransientService)

	if s1.id == s2.id {
		t.Error("transient scope in lazy module: expected different instances")
	}
}
