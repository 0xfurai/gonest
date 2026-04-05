package integration

import (
	"testing"

	"github.com/0xfurai/gonest"
)

// ---------------------------------------------------------------------------
// Lifecycle Hooks Integration Tests
// Mirror: original/integration/hooks/
// ---------------------------------------------------------------------------

type hookTracker struct {
	order []string
}

// ---------------------------------------------------------------------------
// Service implementing all lifecycle hooks
// ---------------------------------------------------------------------------

type allHooksService struct {
	tracker *hookTracker
	name    string
}

func (s *allHooksService) OnModuleInit() error {
	s.tracker.order = append(s.tracker.order, s.name+":OnModuleInit")
	return nil
}

func (s *allHooksService) OnApplicationBootstrap() error {
	s.tracker.order = append(s.tracker.order, s.name+":OnApplicationBootstrap")
	return nil
}

func (s *allHooksService) OnModuleDestroy() error {
	s.tracker.order = append(s.tracker.order, s.name+":OnModuleDestroy")
	return nil
}

func (s *allHooksService) BeforeApplicationShutdown(signal string) error {
	s.tracker.order = append(s.tracker.order, s.name+":BeforeApplicationShutdown")
	return nil
}

func (s *allHooksService) OnApplicationShutdown(signal string) error {
	s.tracker.order = append(s.tracker.order, s.name+":OnApplicationShutdown")
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHooks_LifecycleOrder(t *testing.T) {
	tracker := &hookTracker{}
	svc := &allHooksService{tracker: tracker, name: "svc"}

	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*allHooksService](svc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})

	// Init triggers OnModuleInit + OnApplicationBootstrap
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}

	// Verify init hooks were called in order
	if len(tracker.order) != 2 {
		t.Fatalf("expected 2 hooks after init, got %d: %v", len(tracker.order), tracker.order)
	}
	if tracker.order[0] != "svc:OnModuleInit" {
		t.Errorf("expected OnModuleInit first, got %q", tracker.order[0])
	}
	if tracker.order[1] != "svc:OnApplicationBootstrap" {
		t.Errorf("expected OnApplicationBootstrap second, got %q", tracker.order[1])
	}

	// Close triggers BeforeApplicationShutdown, OnApplicationShutdown, OnModuleDestroy
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"svc:OnModuleInit",
		"svc:OnApplicationBootstrap",
		"svc:BeforeApplicationShutdown",
		"svc:OnApplicationShutdown",
		"svc:OnModuleDestroy",
	}

	if len(tracker.order) != len(expected) {
		t.Fatalf("expected %d hooks, got %d: %v", len(expected), len(tracker.order), tracker.order)
	}
	for i, exp := range expected {
		if tracker.order[i] != exp {
			t.Errorf("hook[%d]: expected %q, got %q", i, exp, tracker.order[i])
		}
	}
}

// parentHookService is a distinct type so it doesn't collide with allHooksService in the container.
type parentHookService struct {
	tracker *hookTracker
	name    string
}

func (s *parentHookService) OnModuleInit() error {
	s.tracker.order = append(s.tracker.order, s.name+":OnModuleInit")
	return nil
}

func (s *parentHookService) OnApplicationBootstrap() error {
	s.tracker.order = append(s.tracker.order, s.name+":OnApplicationBootstrap")
	return nil
}

func TestHooks_MultiModule_Order(t *testing.T) {
	tracker := &hookTracker{}
	childSvc := &allHooksService{tracker: tracker, name: "child"}
	parentSvc := &parentHookService{tracker: tracker, name: "parent"}

	childModule := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*allHooksService](childSvc)},
	})

	parentModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:   []*gonest.Module{childModule},
		Providers: []any{gonest.ProvideValue[*parentHookService](parentSvc)},
	})

	app := gonest.Create(parentModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}

	// Child module compiles first, so child hooks fire before parent
	if len(tracker.order) < 2 {
		t.Fatalf("expected at least 2 hooks, got %d: %v", len(tracker.order), tracker.order)
	}

	// OnModuleInit: child before parent
	if tracker.order[0] != "child:OnModuleInit" {
		t.Errorf("expected child OnModuleInit first, got %q", tracker.order[0])
	}

	app.Close()
}

func TestHooks_OnModuleInit_CalledDuringCompile(t *testing.T) {
	tracker := &hookTracker{}
	svc := &allHooksService{tracker: tracker, name: "svc"}

	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*allHooksService](svc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})

	// Before Init, no hooks should have been called
	if len(tracker.order) != 0 {
		t.Fatalf("expected 0 hooks before init, got %d", len(tracker.order))
	}

	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// OnModuleInit is called during compile (part of Init)
	found := false
	for _, h := range tracker.order {
		if h == "svc:OnModuleInit" {
			found = true
			break
		}
	}
	if !found {
		t.Error("OnModuleInit not called during Init")
	}
}

func TestHooks_OnApplicationBootstrap_AfterAllModulesInit(t *testing.T) {
	tracker := &hookTracker{}
	svc := &allHooksService{tracker: tracker, name: "svc"}

	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*allHooksService](svc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Verify OnApplicationBootstrap was called after OnModuleInit
	initIdx := -1
	bootstrapIdx := -1
	for i, h := range tracker.order {
		if h == "svc:OnModuleInit" {
			initIdx = i
		}
		if h == "svc:OnApplicationBootstrap" {
			bootstrapIdx = i
		}
	}
	if initIdx == -1 || bootstrapIdx == -1 {
		t.Fatal("both hooks should have been called")
	}
	if bootstrapIdx <= initIdx {
		t.Error("OnApplicationBootstrap should be called after OnModuleInit")
	}
}

func TestHooks_ShutdownOrder(t *testing.T) {
	tracker := &hookTracker{}
	svc := &allHooksService{tracker: tracker, name: "svc"}

	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*allHooksService](svc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	app.Close()

	// Extract shutdown hooks
	var shutdownHooks []string
	for _, h := range tracker.order {
		if h == "svc:BeforeApplicationShutdown" || h == "svc:OnApplicationShutdown" || h == "svc:OnModuleDestroy" {
			shutdownHooks = append(shutdownHooks, h)
		}
	}

	// BeforeApplicationShutdown → OnApplicationShutdown → OnModuleDestroy
	if len(shutdownHooks) != 3 {
		t.Fatalf("expected 3 shutdown hooks, got %d: %v", len(shutdownHooks), shutdownHooks)
	}
	if shutdownHooks[0] != "svc:BeforeApplicationShutdown" {
		t.Errorf("expected BeforeApplicationShutdown first, got %q", shutdownHooks[0])
	}
	if shutdownHooks[1] != "svc:OnApplicationShutdown" {
		t.Errorf("expected OnApplicationShutdown second, got %q", shutdownHooks[1])
	}
	if shutdownHooks[2] != "svc:OnModuleDestroy" {
		t.Errorf("expected OnModuleDestroy third, got %q", shutdownHooks[2])
	}
}

// Test that providers without lifecycle hooks don't cause errors
func TestHooks_NoHooksProvider(t *testing.T) {
	type plainService struct{ Value string }
	svc := &plainService{Value: "plain"}

	module := gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*plainService](svc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	if err := app.Close(); err != nil {
		t.Fatal(err)
	}
}
