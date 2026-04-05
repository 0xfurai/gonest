package integration

import (
	"reflect"
	"testing"

	"github.com/gonest"
)

// ---------------------------------------------------------------------------
// Discovery Integration Tests
// Mirror: original/integration/discovery/
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Discoverable services
// ---------------------------------------------------------------------------

type webhookService struct {
	Name string
}

func newWebhookService() *webhookService {
	return &webhookService{Name: "cleanup"}
}

type flushWebhookService struct {
	Name string
}

func newFlushWebhookService() *flushWebhookService {
	return &flushWebhookService{Name: "flush"}
}

// Loggable is an interface for testing GetProvidersWithInterface.
type Loggable interface {
	LogName() string
}

type loggableServiceA struct{}

func newLoggableServiceA() *loggableServiceA { return &loggableServiceA{} }
func (s *loggableServiceA) LogName() string   { return "A" }

type loggableServiceB struct{}

func newLoggableServiceB() *loggableServiceB { return &loggableServiceB{} }
func (s *loggableServiceB) LogName() string   { return "B" }

type nonLoggableService struct{}

func newNonLoggableService() *nonLoggableService { return &nonLoggableService{} }

// ---------------------------------------------------------------------------
// Controller for discovery tests
// ---------------------------------------------------------------------------

type discoveryController struct{}

func newDiscoveryController() *discoveryController { return &discoveryController{} }

func (c *discoveryController) Register(r gonest.Router) {
	r.Get("/discover", func(ctx gonest.Context) error {
		return ctx.JSON(200, "ok")
	})
}

// ---------------------------------------------------------------------------
// Tests: DiscoveryService.GetProviders
// ---------------------------------------------------------------------------

func TestDiscovery_GetProviders_ReturnsAllProviders(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newWebhookService, newFlushWebhookService},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	providers := ds.GetProviders()

	if len(providers) == 0 {
		t.Fatal("expected at least 2 providers")
	}

	typeNames := make(map[string]bool)
	for _, p := range providers {
		typeNames[p.Type.String()] = true
	}

	if !typeNames["*integration.webhookService"] {
		t.Error("expected webhookService in discovered providers")
	}
	if !typeNames["*integration.flushWebhookService"] {
		t.Error("expected flushWebhookService in discovered providers")
	}
}

func TestDiscovery_GetProviders_IncludesImportedModules(t *testing.T) {
	childModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newWebhookService},
		Exports:   []any{(*webhookService)(nil)},
	})

	parentModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{childModule},
		Providers: []any{newFlushWebhookService},
	})

	app := gonest.Create(parentModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	providers := ds.GetProviders()

	typeNames := make(map[string]bool)
	for _, p := range providers {
		typeNames[p.Type.String()] = true
	}

	if !typeNames["*integration.webhookService"] {
		t.Error("expected webhookService from child module")
	}
	if !typeNames["*integration.flushWebhookService"] {
		t.Error("expected flushWebhookService from parent module")
	}
}

// ---------------------------------------------------------------------------
// Tests: DiscoveryService.GetControllers
// ---------------------------------------------------------------------------

func TestDiscovery_GetControllers_ReturnsAllControllers(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newDiscoveryController},
		Providers:   []any{newWebhookService},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	controllers := ds.GetControllers()

	if len(controllers) == 0 {
		t.Fatal("expected at least 1 controller")
	}

	found := false
	for _, c := range controllers {
		if reflect.TypeOf(c.Instance).String() == "*integration.discoveryController" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected discoveryController in discovered controllers")
	}
}

func TestDiscovery_GetControllers_AcrossModules(t *testing.T) {
	childModule := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newDiscoveryController},
	})

	parentModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{childModule},
	})

	app := gonest.Create(parentModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	controllers := ds.GetControllers()

	if len(controllers) == 0 {
		t.Fatal("expected controller from child module")
	}
}

// ---------------------------------------------------------------------------
// Tests: GetProvidersWithInterface
// ---------------------------------------------------------------------------

func TestDiscovery_GetProvidersWithInterface(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			newLoggableServiceA,
			newLoggableServiceB,
			newNonLoggableService,
		},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	loggables := gonest.GetProvidersWithInterface[Loggable](ds)

	if len(loggables) != 2 {
		t.Fatalf("expected 2 Loggable providers, got %d", len(loggables))
	}

	names := make(map[string]bool)
	for _, l := range loggables {
		names[l.LogName()] = true
	}

	if !names["A"] || !names["B"] {
		t.Errorf("expected loggables A and B, got %v", names)
	}
}

func TestDiscovery_GetProvidersWithInterface_NoMatch(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newNonLoggableService},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	loggables := gonest.GetProvidersWithInterface[Loggable](ds)

	if len(loggables) != 0 {
		t.Errorf("expected 0 Loggable providers, got %d", len(loggables))
	}
}

// ---------------------------------------------------------------------------
// Tests: DiscoveryModule as importable module
// ---------------------------------------------------------------------------

type discoveryConsumer struct {
	ds *gonest.DiscoveryService
}

func newDiscoveryConsumer(ds *gonest.DiscoveryService) *discoveryConsumer {
	return &discoveryConsumer{ds: ds}
}

func TestDiscovery_DiscoveryModule_ProvidesViaImport(t *testing.T) {
	discoveryMod := gonest.NewDiscoveryModule()

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{discoveryMod},
		Providers: []any{newDiscoveryConsumer, newWebhookService},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	consumer, err := gonest.Resolve[*discoveryConsumer](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if consumer.ds == nil {
		t.Fatal("expected DiscoveryService to be injected")
	}
}

func TestDiscovery_DiscoveryModule_IsGlobal(t *testing.T) {
	discoveryMod := gonest.NewDiscoveryModule()

	// Root module imports DiscoveryModule (global) and uses its exports directly
	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{discoveryMod},
		Providers: []any{newDiscoveryConsumer},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	consumer, err := gonest.Resolve[*discoveryConsumer](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if consumer.ds == nil {
		t.Fatal("expected DiscoveryService from global DiscoveryModule")
	}
}

// ---------------------------------------------------------------------------
// Tests: GraphInspector provided via DiscoveryModule
// ---------------------------------------------------------------------------

type inspectorConsumer struct {
	gi *gonest.GraphInspector
}

func newInspectorConsumer(gi *gonest.GraphInspector) *inspectorConsumer {
	return &inspectorConsumer{gi: gi}
}

func TestDiscovery_DiscoveryModule_ProvidesGraphInspector(t *testing.T) {
	discoveryMod := gonest.NewDiscoveryModule()

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{discoveryMod},
		Providers: []any{newInspectorConsumer},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	consumer, err := gonest.Resolve[*inspectorConsumer](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if consumer.gi == nil {
		t.Fatal("expected GraphInspector to be injected")
	}
}

// ---------------------------------------------------------------------------
// Tests: Provider host module tracking
// ---------------------------------------------------------------------------

func TestDiscovery_ProviderHostModule(t *testing.T) {
	childModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{newWebhookService},
		Exports:   []any{(*webhookService)(nil)},
	})

	parentModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{childModule},
		Providers: []any{newFlushWebhookService},
	})

	app := gonest.Create(parentModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	ds := app.GetDiscoveryService()
	providers := ds.GetProviders()

	for _, p := range providers {
		if p.Module == nil {
			t.Errorf("provider %s has nil Module", p.Type.String())
		}
	}
}
