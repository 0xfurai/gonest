package gonest

import "testing"

func TestCreateApplicationContext(t *testing.T) {
	module := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
		Exports:   []any{(*greetingService)(nil)},
	})

	ctx, err := CreateApplicationContext(module, ApplicationOptions{Logger: NopLogger{}})
	if err != nil {
		t.Fatalf("create context failed: %v", err)
	}

	container := ctx.GetContainer()
	svc, err := Resolve[*greetingService](container)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if svc.greeting != "Hello, World!" {
		t.Errorf("expected greeting, got %q", svc.greeting)
	}

	if err := ctx.Close(); err != nil {
		t.Errorf("close failed: %v", err)
	}
}

func TestCreateApplicationContext_LifecycleHooks(t *testing.T) {
	svc := newLifecycleService()
	module := NewModule(ModuleOptions{
		Providers: []any{ProvideValue[*lifecycleService](svc)},
	})

	ctx, err := CreateApplicationContext(module, ApplicationOptions{Logger: NopLogger{}})
	if err != nil {
		t.Fatalf("create context failed: %v", err)
	}

	if !svc.initCalled {
		t.Error("expected OnModuleInit to be called")
	}
	if !svc.bootstrapCalled {
		t.Error("expected OnApplicationBootstrap to be called")
	}

	ctx.Close()
	if !svc.destroyCalled {
		t.Error("expected OnModuleDestroy to be called")
	}
}

func TestCreateApplicationContext_DiscoveryService(t *testing.T) {
	module := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
	})

	ctx, err := CreateApplicationContext(module, ApplicationOptions{Logger: NopLogger{}})
	if err != nil {
		t.Fatalf("create context failed: %v", err)
	}
	defer ctx.Close()

	ds := ctx.GetDiscoveryService()
	if ds == nil {
		t.Fatal("expected non-nil discovery service")
	}

	providers := ds.GetProviders()
	if len(providers) == 0 {
		t.Error("expected at least one provider")
	}
}
