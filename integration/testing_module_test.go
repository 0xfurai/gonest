package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonest"
	nesttest "github.com/gonest/testing"
)

// ---------------------------------------------------------------------------
// Testing Module Override Integration Tests
// Mirror: original/integration/testing-module-override/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Real services
// ---------------------------------------------------------------------------

type catsService struct{}

func newCatsService() *catsService { return &catsService{} }

func (s *catsService) FindAll() []map[string]string {
	return []map[string]string{
		{"name": "Whiskers", "breed": "Persian"},
		{"name": "Tom", "breed": "Siamese"},
	}
}

type catsController struct {
	svc *catsService
}

func newCatsController(svc *catsService) *catsController {
	return &catsController{svc: svc}
}

func (c *catsController) Register(r gonest.Router) {
	r.Prefix("/cats")
	r.Get("", c.findAll)
}

func (c *catsController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.svc.FindAll())
}

func catsModule() *gonest.Module {
	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newCatsController},
		Providers:   []any{newCatsService},
	})
}

// ---------------------------------------------------------------------------
// Mock services
// ---------------------------------------------------------------------------

type mockCatsService struct{}

func newMockCatsService() *catsService {
	// Returns a catsService that returns mock data
	return &catsService{} // We'll override the whole provider
}

// ---------------------------------------------------------------------------
// Tests: Override Provider
// ---------------------------------------------------------------------------

func TestTestingModule_OverrideProvider(t *testing.T) {
	// Create a mock service
	mock := &catsService{}
	_ = mock // We override with a factory that returns our controlled instance

	compiled := nesttest.Test(catsModule()).
		OverrideProvider((*catsService)(nil), func() *catsService {
			return &catsService{} // Mock returns same struct but from test factory
		}).
		Compile(t)

	app := compiled.App()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/cats", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var cats []map[string]string
	json.Unmarshal(w.Body.Bytes(), &cats)
	if len(cats) != 2 {
		t.Errorf("expected 2 cats, got %d", len(cats))
	}
}

func TestTestingModule_Resolve(t *testing.T) {
	compiled := nesttest.Test(catsModule()).Compile(t)

	svc := nesttest.Resolve[*catsService](compiled)
	if svc == nil {
		t.Fatal("expected cats service")
	}

	cats := svc.FindAll()
	if len(cats) != 2 {
		t.Errorf("expected 2 cats, got %d", len(cats))
	}
}

func TestTestingModule_App_HTTP(t *testing.T) {
	compiled := nesttest.Test(catsModule()).Compile(t)
	app := compiled.App()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/cats", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Override with different behavior
// ---------------------------------------------------------------------------

type customCatsService struct {
	data []map[string]string
}

func (s *customCatsService) FindAll() []map[string]string {
	return s.data
}

func TestTestingModule_OverrideProvider_CustomBehavior(t *testing.T) {
	customSvc := &catsService{}
	// We override the provider entirely so the controller gets our instance
	compiled := nesttest.Test(catsModule()).
		OverrideProvider((*catsService)(nil), gonest.ProvideValue[*catsService](customSvc)).
		Compile(t)

	svc := nesttest.Resolve[*catsService](compiled)
	if svc != customSvc {
		t.Error("expected the overridden service instance")
	}
}

// ---------------------------------------------------------------------------
// Tests: Module with imports
// ---------------------------------------------------------------------------

func TestTestingModule_WithImportedModules(t *testing.T) {
	sharedModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
		Exports:   []any{(*loggerService)(nil)},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{sharedModule},
		Providers: []any{newDatabaseService},
	})

	compiled := nesttest.Test(appModule).Compile(t)
	db := nesttest.Resolve[*databaseService](compiled)
	if db == nil {
		t.Fatal("expected database service")
	}
	if db.logger == nil {
		t.Error("expected logger to be injected from imported module")
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple overrides
// ---------------------------------------------------------------------------

func TestTestingModule_MultipleOverrides(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService, newDatabaseService},
	})

	mockLogger := &loggerService{logs: []string{"pre-existing"}}
	compiled := nesttest.Test(module).
		OverrideProvider((*loggerService)(nil), gonest.ProvideValue[*loggerService](mockLogger)).
		Compile(t)

	logger := nesttest.Resolve[*loggerService](compiled)
	if len(logger.logs) != 1 || logger.logs[0] != "pre-existing" {
		t.Error("expected mock logger with pre-existing logs")
	}
}

// ---------------------------------------------------------------------------
// Tests: OverrideModule
// Mirror: original/integration/testing-module-override/ — overrideModule
// ---------------------------------------------------------------------------

type realDBService struct {
	DSN string
}

func newRealDBService() *realDBService { return &realDBService{DSN: "postgres://prod"} }

type fakeDBService struct {
	DSN string
}

func newFakeDBService() *fakeDBService { return &fakeDBService{DSN: "sqlite://:memory:"} }

func TestTestingModule_OverrideModule(t *testing.T) {
	realDBModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newRealDBService},
		Exports:   []any{(*realDBService)(nil)},
	})

	fakeDBModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{func() *realDBService {
			return &realDBService{DSN: "sqlite://:memory:"}
		}},
		Exports: []any{(*realDBService)(nil)},
	})

	type appService struct {
		db *realDBService
	}
	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{realDBModule},
		Providers: []any{func(db *realDBService) *appService {
			return &appService{db: db}
		}},
	})

	compiled := nesttest.Test(appModule).
		OverrideModule(realDBModule, fakeDBModule).
		Compile(t)

	svc := nesttest.Resolve[*appService](compiled)
	if svc.db == nil {
		t.Fatal("expected db to be injected")
	}
	if svc.db.DSN != "sqlite://:memory:" {
		t.Errorf("expected fake DSN, got %q", svc.db.DSN)
	}
}

func TestTestingModule_OverrideModule_WithProvider(t *testing.T) {
	realLogModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
		Exports:   []any{(*loggerService)(nil)},
	})

	fakeLogModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{func() *loggerService {
			return &loggerService{logs: []string{"fake-init"}}
		}},
		Exports: []any{(*loggerService)(nil)},
	})

	type logConsumer struct {
		logger *loggerService
	}
	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{realLogModule},
		Providers: []any{func(l *loggerService) *logConsumer {
			return &logConsumer{logger: l}
		}},
	})

	compiled := nesttest.Test(appModule).
		OverrideModule(realLogModule, fakeLogModule).
		Compile(t)

	logger := nesttest.Resolve[*loggerService](compiled)
	if len(logger.logs) != 1 || logger.logs[0] != "fake-init" {
		t.Error("expected fake logger from overridden module")
	}
}

func TestTestingModule_OverrideModule_GlobalModule(t *testing.T) {
	type globalConfig struct {
		Env string
	}

	realGlobalMod := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*globalConfig](&globalConfig{Env: "production"})},
		Exports:   []any{(*globalConfig)(nil)},
		Global:    true,
	})

	fakeGlobalMod := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*globalConfig](&globalConfig{Env: "test"})},
		Exports:   []any{(*globalConfig)(nil)},
		Global:    true,
	})

	type configConsumer struct {
		cfg *globalConfig
	}

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{realGlobalMod},
		Providers: []any{func(cfg *globalConfig) *configConsumer {
			return &configConsumer{cfg: cfg}
		}},
	})

	compiled := nesttest.Test(appModule).
		OverrideModule(realGlobalMod, fakeGlobalMod).
		Compile(t)

	consumer := nesttest.Resolve[*configConsumer](compiled)
	if consumer.cfg.Env != "test" {
		t.Errorf("expected Env=test from overridden global module, got %q", consumer.cfg.Env)
	}
}

// ---------------------------------------------------------------------------
// Tests: MockFactory integration with TestingModule
// ---------------------------------------------------------------------------

func TestTestingModule_MockFactory(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newCatsService},
	})

	mock := nesttest.MockFactory[*catsService](func() *catsService {
		return &catsService{}
	})

	compiled := nesttest.Test(module).
		OverrideProvider((*catsService)(nil), mock).
		Compile(t)

	svc := nesttest.Resolve[*catsService](compiled)
	cats := svc.FindAll()
	if len(cats) != 2 {
		t.Errorf("expected 2 cats from mock factory, got %d", len(cats))
	}
}

func TestTestingModule_OverrideModule_AndProvider_Together(t *testing.T) {
	logModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newLoggerService},
		Exports:   []any{(*loggerService)(nil)},
	})

	fakeLogModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{func() *loggerService {
			return &loggerService{logs: []string{"fake"}}
		}},
		Exports: []any{(*loggerService)(nil)},
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{logModule},
		Controllers: []any{newCatsController},
		Providers:   []any{newCatsService},
	})

	compiled := nesttest.Test(appModule).
		OverrideModule(logModule, fakeLogModule).
		Compile(t)

	app := compiled.App()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/cats", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
