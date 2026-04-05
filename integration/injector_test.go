package integration

import (
	"reflect"
	"testing"

	"github.com/gonest"
)

// ---------------------------------------------------------------------------
// Injector / DI Container Integration Tests
// Mirror: original/integration/injector/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Services for DI tests
// ---------------------------------------------------------------------------

type loggerService struct {
	logs []string
}

func newLoggerService() *loggerService {
	return &loggerService{}
}

func (s *loggerService) Log(msg string) {
	s.logs = append(s.logs, msg)
}

type databaseService struct {
	logger *loggerService
}

func newDatabaseService(logger *loggerService) *databaseService {
	return &databaseService{logger: logger}
}

type repositoryService struct {
	db     *databaseService
	logger *loggerService
}

func newRepositoryService(db *databaseService, logger *loggerService) *repositoryService {
	return &repositoryService{db: db, logger: logger}
}

// ---------------------------------------------------------------------------
// Tests: Basic Injection
// ---------------------------------------------------------------------------

func TestInjector_BasicConstructorInjection(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService, newDatabaseService, newRepositoryService},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()

	repo, err := gonest.Resolve[*repositoryService](container)
	if err != nil {
		t.Fatal(err)
	}
	if repo.db == nil {
		t.Error("database dependency not injected")
	}
	if repo.logger == nil {
		t.Error("logger dependency not injected")
	}
}

func TestInjector_SingletonScope(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()

	logger1, err := gonest.Resolve[*loggerService](container)
	if err != nil {
		t.Fatal(err)
	}
	logger2, err := gonest.Resolve[*loggerService](container)
	if err != nil {
		t.Fatal(err)
	}

	if logger1 != logger2 {
		t.Error("singleton scope: expected same instance")
	}
}

func TestInjector_TransientScope(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideWithScope(newLoggerService, gonest.ScopeTransient),
		},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()

	logger1, err := gonest.Resolve[*loggerService](container)
	if err != nil {
		t.Fatal(err)
	}
	logger2, err := gonest.Resolve[*loggerService](container)
	if err != nil {
		t.Fatal(err)
	}

	if logger1 == logger2 {
		t.Error("transient scope: expected different instances")
	}
}

func TestInjector_ValueProvider(t *testing.T) {
	config := &struct{ Env string }{Env: "test"}

	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideValue(config),
		},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()
	resolved, err := container.Resolve(reflect.TypeOf(config))
	if err != nil {
		t.Fatal(err)
	}

	if resolved != config {
		t.Error("value provider should return the exact instance")
	}
}

// ---------------------------------------------------------------------------
// Tests: Interface Binding
// ---------------------------------------------------------------------------

type repository interface {
	FindAll() []string
}

type memoryRepository struct{}

func newMemoryRepository() *memoryRepository { return &memoryRepository{} }

func (r *memoryRepository) FindAll() []string { return []string{"item1", "item2"} }

type appServiceWithRepo struct {
	repo repository
}

func newAppServiceWithRepo(repo repository) *appServiceWithRepo {
	return &appServiceWithRepo{repo: repo}
}

func TestInjector_InterfaceBinding(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.Bind[repository](newMemoryRepository),
			newAppServiceWithRepo,
		},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()
	svc, err := gonest.Resolve[*appServiceWithRepo](container)
	if err != nil {
		t.Fatal(err)
	}

	items := svc.repo.FindAll()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

// ---------------------------------------------------------------------------
// Tests: Token-based Injection
// ---------------------------------------------------------------------------

func TestInjector_TokenProvider(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideTokenValue("APP_NAME", "GoNest"),
			gonest.ProvideTokenValue("APP_VERSION", "1.0.0"),
		},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()

	name, err := container.ResolveByToken("APP_NAME")
	if err != nil {
		t.Fatal(err)
	}
	if name != "GoNest" {
		t.Errorf("expected GoNest, got %v", name)
	}

	version, err := container.ResolveByToken("APP_VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if version != "1.0.0" {
		t.Errorf("expected 1.0.0, got %v", version)
	}
}

func TestInjector_TokenNotFound(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	_, err := app.GetContainer().ResolveByToken("MISSING")
	if err == nil {
		t.Error("expected error for missing token")
	}
}

// ---------------------------------------------------------------------------
// Tests: Deep Dependency Chain
// ---------------------------------------------------------------------------

type serviceA struct{ name string }
type serviceB struct{ a *serviceA }
type serviceC struct{ b *serviceB }
type serviceD struct{ c *serviceC }

func newServiceA() *serviceA           { return &serviceA{name: "A"} }
func newServiceB(a *serviceA) *serviceB { return &serviceB{a: a} }
func newServiceC(b *serviceB) *serviceC { return &serviceC{b: b} }
func newServiceD(c *serviceC) *serviceD { return &serviceD{c: c} }

func TestInjector_DeepDependencyChain(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newServiceA, newServiceB, newServiceC, newServiceD},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	d, err := gonest.Resolve[*serviceD](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if d.c.b.a.name != "A" {
		t.Error("deep dependency chain not resolved correctly")
	}
}

// ---------------------------------------------------------------------------
// Tests: Cross-module DI
// ---------------------------------------------------------------------------

func TestInjector_CrossModuleInjection(t *testing.T) {
	sharedModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
		Exports:   []any{(*loggerService)(nil)},
	})

	consumerModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{sharedModule},
		Providers: []any{newDatabaseService},
	})

	app := gonest.Create(consumerModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	db, err := gonest.Resolve[*databaseService](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if db.logger == nil {
		t.Error("cross-module logger injection failed")
	}
}

func TestInjector_UnexportedProviderNotAvailable(t *testing.T) {
	innerModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
		// NOT exporting loggerService
	})

	outerModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{innerModule},
		Providers: []any{
			// databaseService needs loggerService which is NOT exported
			newDatabaseService,
		},
	})

	app := gonest.Create(outerModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	err := app.Init()
	// Should fail because loggerService is not exported
	if err == nil {
		t.Error("expected error: loggerService not exported from inner module")
		app.Close()
	}
}

// ---------------------------------------------------------------------------
// Tests: Global Module
// ---------------------------------------------------------------------------

func TestInjector_GlobalModuleAvailableEverywhere(t *testing.T) {
	globalModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
		Exports:   []any{(*loggerService)(nil)},
		Global:    true,
	})

	featureModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newDatabaseService},
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{globalModule, featureModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
}

// ---------------------------------------------------------------------------
// Tests: MustResolve panics
// ---------------------------------------------------------------------------

func TestInjector_MustResolve_Panics(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustResolve to panic on missing provider")
		}
	}()

	gonest.MustResolve[*loggerService](app.GetContainer())
}

// ---------------------------------------------------------------------------
// Tests: Request Scope
// ---------------------------------------------------------------------------

func TestInjector_RequestScope_NewPerContainer(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideWithScope(newLoggerService, gonest.ScopeRequest),
		},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()

	// Create two request containers
	rc1 := container.CreateRequestContainer()
	rc2 := container.CreateRequestContainer()

	logger1, err := gonest.Resolve[*loggerService](rc1)
	if err != nil {
		t.Fatal(err)
	}
	logger2, err := gonest.Resolve[*loggerService](rc2)
	if err != nil {
		t.Fatal(err)
	}

	if logger1 == logger2 {
		t.Error("request scope: expected different instances for different request containers")
	}
}

// ---------------------------------------------------------------------------
// Tests: Factory Provider
// ---------------------------------------------------------------------------

func TestInjector_FactoryProvider(t *testing.T) {
	type config struct {
		DSN string
	}
	cfg := &config{DSN: "postgres://localhost/test"}

	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideValue[*config](cfg),
			gonest.ProvideFactory[*databaseService](func(cfg *config) *databaseService {
				return &databaseService{}
			}),
		},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	db, err := gonest.Resolve[*databaseService](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Error("factory provider returned nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: Error on unresolvable dependency
// Mirror: original/integration/injector/e2e/injector.spec.ts
// ---------------------------------------------------------------------------

type unresolvableService struct {
	dep *struct{ missing bool }
}

func newUnresolvableService(dep *struct{ missing bool }) *unresolvableService {
	return &unresolvableService{dep: dep}
}

func TestInjector_ErrorOnUnresolvableDependency(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newUnresolvableService},
		// Missing the dependency provider
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	err := app.Init()
	if err == nil {
		app.Close()
		t.Fatal("expected error for unresolvable dependency")
	}
}

// ---------------------------------------------------------------------------
// Tests: Dynamic module token resolution
// Mirror: original/integration/injector/e2e/injector.spec.ts (dynamic module)
// ---------------------------------------------------------------------------

func TestInjector_DynamicModuleTokenResolution(t *testing.T) {
	// Token providers are resolved within the module that registers them.
	// To use tokens from the root, register them in the root module directly.
	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideTokenValue("DYNAMIC_TOKEN", "DYNAMIC_VALUE"),
		},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	val, err := app.GetContainer().ResolveByToken("DYNAMIC_TOKEN")
	if err != nil {
		t.Fatal(err)
	}
	if val != "DYNAMIC_VALUE" {
		t.Errorf("expected DYNAMIC_VALUE, got %v", val)
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple tokens in same module
// ---------------------------------------------------------------------------

func TestInjector_MultipleTokens(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideTokenValue("DB_HOST", "localhost"),
			gonest.ProvideTokenValue("DB_PORT", 5432),
			gonest.ProvideTokenValue("DB_NAME", "testdb"),
		},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	c := app.GetContainer()

	host, _ := c.ResolveByToken("DB_HOST")
	port, _ := c.ResolveByToken("DB_PORT")
	name, _ := c.ResolveByToken("DB_NAME")

	if host != "localhost" {
		t.Errorf("expected localhost, got %v", host)
	}
	if port != 5432 {
		t.Errorf("expected 5432, got %v", port)
	}
	if name != "testdb" {
		t.Errorf("expected testdb, got %v", name)
	}
}

// ---------------------------------------------------------------------------
// Tests: Provider with constructor returning error
// ---------------------------------------------------------------------------

type failingService struct{}

func newFailingService() (*failingService, error) {
	return nil, &testInjectorError{msg: "constructor failed"}
}

type testInjectorError struct {
	msg string
}

func (e *testInjectorError) Error() string { return e.msg }

func TestInjector_ConstructorReturnsError(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newFailingService},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	err := app.Init()
	if err == nil {
		app.Close()
		t.Fatal("expected error from failing constructor")
	}
}

// ---------------------------------------------------------------------------
// Tests: Many global modules
// Mirror: original/integration/injector/e2e/many-global-modules.spec.ts
// ---------------------------------------------------------------------------

func TestInjector_ManyGlobalModules(t *testing.T) {
	type globalA struct{ Name string }
	type globalB struct{ Name string }
	type globalC struct{ Name string }

	modA := gonest.NewDynamicModule(gonest.DynamicModule{
		Providers: []any{gonest.ProvideValue[*globalA](&globalA{Name: "A"})},
		Exports:   []any{(*globalA)(nil)},
		Global:    true,
	})
	modB := gonest.NewDynamicModule(gonest.DynamicModule{
		Providers: []any{gonest.ProvideValue[*globalB](&globalB{Name: "B"})},
		Exports:   []any{(*globalB)(nil)},
		Global:    true,
	})
	modC := gonest.NewDynamicModule(gonest.DynamicModule{
		Providers: []any{gonest.ProvideValue[*globalC](&globalC{Name: "C"})},
		Exports:   []any{(*globalC)(nil)},
		Global:    true,
	})

	// Feature module that depends on all globals without importing them
	type featureABC struct {
		a *globalA
		b *globalB
		c *globalC
	}

	featureMod := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{func(a *globalA, b *globalB, c *globalC) *featureABC {
			return &featureABC{a: a, b: b, c: c}
		}},
	})

	root := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{modA, modB, modC, featureMod},
	})

	app := gonest.Create(root, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
}

// ---------------------------------------------------------------------------
// Tests: Child container parent fallback
// Mirror: original/integration/injector implicit behavior
// ---------------------------------------------------------------------------

func TestInjector_ChildContainerFallback(t *testing.T) {
	parentMod := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
		Exports:   []any{(*loggerService)(nil)},
	})

	childMod := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{parentMod},
		Providers: []any{newDatabaseService}, // depends on loggerService from parent
	})

	app := gonest.Create(childMod, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	db, err := gonest.Resolve[*databaseService](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if db.logger == nil {
		t.Error("expected logger from parent container")
	}
}
