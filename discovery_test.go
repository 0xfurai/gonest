package gonest

import "testing"

func TestDiscoveryService_GetProviders(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ds := app.GetDiscoveryService()
	providers := ds.GetProviders()
	if len(providers) == 0 {
		t.Fatal("expected at least one provider")
	}

	found := false
	for _, p := range providers {
		if _, ok := p.Instance.(*greetingService); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find greetingService in providers")
	}
}

func TestDiscoveryService_GetControllers(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ds := app.GetDiscoveryService()
	controllers := ds.GetControllers()
	if len(controllers) == 0 {
		t.Fatal("expected at least one controller")
	}
}

func TestGraphInspector_GetModules(t *testing.T) {
	child := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
		Exports:   []any{(*greetingService)(nil)},
	})
	root := NewModule(ModuleOptions{
		Imports:     []*Module{child},
		Controllers: []any{newGreetingController},
	})

	app := Create(root, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	gi := app.GetGraphInspector()
	modules := gi.GetModules()
	if len(modules) < 2 {
		t.Errorf("expected at least 2 modules, got %d", len(modules))
	}
}

func TestGraphInspector_GetDependencies(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	gi := app.GetGraphInspector()
	edges := gi.GetAllDependencies()
	// greetingController depends on greetingService
	found := false
	for _, e := range edges {
		if e.Source.String() == "*gonest.greetingController" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find greetingController dependency edge")
	}
}
