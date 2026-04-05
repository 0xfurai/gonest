package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonest"
)

// ---------------------------------------------------------------------------
// Dynamic Module / Module Utils Integration Tests
// Mirror: original/integration/module-utils/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Database module with ForRoot pattern
// ---------------------------------------------------------------------------

type dbConfig struct {
	DSN      string
	MaxConns int
}

type dbConnection struct {
	config *dbConfig
}

func createDatabaseModule(cfg dbConfig) *gonest.Module {
	conn := &dbConnection{config: &cfg}
	return gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideValue[*dbConfig](&cfg),
			gonest.ProvideValue[*dbConnection](conn),
		},
		Exports: []any{(*dbConfig)(nil), (*dbConnection)(nil)},
		Global:  true,
	})
}

// ---------------------------------------------------------------------------
// Config module with ForRoot pattern using ForRoot helper
// ---------------------------------------------------------------------------

type appConfig struct {
	AppName    string
	AppVersion string
}

func configModuleForRoot(cfg appConfig) *gonest.Module {
	return gonest.ForRoot(cfg, func(c appConfig) *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{gonest.ProvideValue[*appConfig](&c)},
			Exports:   []any{(*appConfig)(nil)},
			Global:    true,
		})
	})
}

// ---------------------------------------------------------------------------
// Feature module with ForFeature pattern
// ---------------------------------------------------------------------------

type featureOptions struct {
	FeatureName string
	Enabled     bool
}

type featureService struct {
	opts *featureOptions
}

func newFeatureService(opts *featureOptions) *featureService {
	return &featureService{opts: opts}
}

func featureModuleForFeature(opts featureOptions) *gonest.Module {
	return gonest.ForFeature(opts, func(o featureOptions) *gonest.Module {
		return gonest.NewModule(gonest.ModuleOptions{
			Providers: []any{
				gonest.ProvideValue[*featureOptions](&o),
				newFeatureService,
			},
			Exports: []any{(*featureService)(nil)},
		})
	})
}

// ---------------------------------------------------------------------------
// Controller using dynamic modules
// ---------------------------------------------------------------------------

type dynamicController struct {
	cfg  *appConfig
	db   *dbConnection
}

func newDynamicController(cfg *appConfig, db *dbConnection) *dynamicController {
	return &dynamicController{cfg: cfg, db: db}
}

func (c *dynamicController) Register(r gonest.Router) {
	r.Get("/config", c.getConfig)
	r.Get("/db", c.getDB)
}

func (c *dynamicController) getConfig(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{
		"appName":    c.cfg.AppName,
		"appVersion": c.cfg.AppVersion,
	})
}

func (c *dynamicController) getDB(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{
		"dsn":      c.db.config.DSN,
		"maxConns": c.db.config.MaxConns,
	})
}

// ---------------------------------------------------------------------------
// Tests: ForRoot pattern
// ---------------------------------------------------------------------------

func TestDynamicModule_ForRoot(t *testing.T) {
	configMod := configModuleForRoot(appConfig{
		AppName:    "GoNest",
		AppVersion: "1.0.0",
	})
	dbMod := createDatabaseModule(dbConfig{
		DSN:      "postgres://localhost/test",
		MaxConns: 10,
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{configMod, dbMod},
		Controllers: []any{newDynamicController},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Test config endpoint
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/config", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var cfgBody map[string]string
	json.Unmarshal(w.Body.Bytes(), &cfgBody)
	if cfgBody["appName"] != "GoNest" {
		t.Errorf("expected GoNest, got %q", cfgBody["appName"])
	}
	if cfgBody["appVersion"] != "1.0.0" {
		t.Errorf("expected 1.0.0, got %q", cfgBody["appVersion"])
	}

	// Test db endpoint
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/db", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var dbBody map[string]any
	json.Unmarshal(w.Body.Bytes(), &dbBody)
	if dbBody["dsn"] != "postgres://localhost/test" {
		t.Errorf("expected dsn, got %v", dbBody["dsn"])
	}
	if dbBody["maxConns"] != float64(10) {
		t.Errorf("expected maxConns=10, got %v", dbBody["maxConns"])
	}
}

// ---------------------------------------------------------------------------
// Tests: ForFeature pattern
// ---------------------------------------------------------------------------

func TestDynamicModule_ForFeature(t *testing.T) {
	featureMod := featureModuleForFeature(featureOptions{
		FeatureName: "caching",
		Enabled:     true,
	})

	featureCtrl := &featureController{}
	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{featureMod},
		Controllers: []any{func(svc *featureService) *featureController {
			featureCtrl.svc = svc
			return featureCtrl
		}},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/feature", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["name"] != "caching" {
		t.Errorf("expected caching, got %v", body["name"])
	}
	if body["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", body["enabled"])
	}
}

type featureController struct {
	svc *featureService
}

func (c *featureController) Register(r gonest.Router) {
	r.Get("/feature", c.handler)
}

func (c *featureController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{
		"name":    c.svc.opts.FeatureName,
		"enabled": c.svc.opts.Enabled,
	})
}

// ---------------------------------------------------------------------------
// Tests: NewDynamicModule
// ---------------------------------------------------------------------------

func TestDynamicModule_NewDynamicModule(t *testing.T) {
	type emailConfig struct {
		Host string
		Port int
	}

	cfg := &emailConfig{Host: "smtp.example.com", Port: 587}

	emailModule := gonest.NewDynamicModule(gonest.DynamicModule{
		Providers: []any{gonest.ProvideValue[*emailConfig](cfg)},
		Exports:   []any{(*emailConfig)(nil)},
		Global:    true,
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{emailModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	resolved, err := gonest.Resolve[*emailConfig](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Host != "smtp.example.com" {
		t.Errorf("expected smtp.example.com, got %q", resolved.Host)
	}
	if resolved.Port != 587 {
		t.Errorf("expected 587, got %d", resolved.Port)
	}
}

// ---------------------------------------------------------------------------
// Tests: Dynamic module is global
// ---------------------------------------------------------------------------

func TestDynamicModule_GlobalExport(t *testing.T) {
	type sharedConfig struct {
		Secret string
	}

	sharedModule := gonest.NewDynamicModule(gonest.DynamicModule{
		Providers: []any{gonest.ProvideValue[*sharedConfig](&sharedConfig{Secret: "s3cr3t"})},
		Exports:   []any{(*sharedConfig)(nil)},
		Global:    true,
	})

	// Feature module doesn't import sharedModule but should still get it
	type featureWithConfig struct {
		config *sharedConfig
	}

	featureMod := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{func(cfg *sharedConfig) *featureWithConfig {
			return &featureWithConfig{config: cfg}
		}},
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{sharedModule, featureMod},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()
}

// ---------------------------------------------------------------------------
// Tests: Multiple dynamic modules
// ---------------------------------------------------------------------------

func TestDynamicModule_MultipleDynamicModules(t *testing.T) {
	type cacheConfig struct {
		TTL int
	}
	type logConfig struct {
		Level string
	}

	cacheMod := gonest.NewDynamicModule(gonest.DynamicModule{
		Providers: []any{gonest.ProvideValue[*cacheConfig](&cacheConfig{TTL: 300})},
		Exports:   []any{(*cacheConfig)(nil)},
		Global:    true,
	})

	logMod := gonest.NewDynamicModule(gonest.DynamicModule{
		Providers: []any{gonest.ProvideValue[*logConfig](&logConfig{Level: "debug"})},
		Exports:   []any{(*logConfig)(nil)},
		Global:    true,
	})

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{cacheMod, logMod},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	container := app.GetContainer()

	cache, err := gonest.Resolve[*cacheConfig](container)
	if err != nil {
		t.Fatal(err)
	}
	if cache.TTL != 300 {
		t.Errorf("expected TTL=300, got %d", cache.TTL)
	}

	logCfg, err := gonest.Resolve[*logConfig](container)
	if err != nil {
		t.Fatal(err)
	}
	if logCfg.Level != "debug" {
		t.Errorf("expected Level=debug, got %q", logCfg.Level)
	}
}
