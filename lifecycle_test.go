package gonest

import "testing"

type lifecycleTracker struct {
	initCalled      bool
	destroyCalled   bool
	bootstrapCalled bool
	shutdownCalled  bool
	beforeShutdown  bool
}

func (l *lifecycleTracker) OnModuleInit() error {
	l.initCalled = true
	return nil
}

func (l *lifecycleTracker) OnModuleDestroy() error {
	l.destroyCalled = true
	return nil
}

func (l *lifecycleTracker) OnApplicationBootstrap() error {
	l.bootstrapCalled = true
	return nil
}

func (l *lifecycleTracker) OnApplicationShutdown(signal string) error {
	l.shutdownCalled = true
	return nil
}

func (l *lifecycleTracker) BeforeApplicationShutdown(signal string) error {
	l.beforeShutdown = true
	return nil
}

func TestLifecycleHooks_FullCycle(t *testing.T) {
	tracker := &lifecycleTracker{}

	dummyCtrl := &struct{ lifecycleController }{}
	module := NewModule(ModuleOptions{
		Controllers: []any{func() *struct{ lifecycleController } { return dummyCtrl }},
		Providers:   []any{ProvideValue[*lifecycleTracker](tracker)},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if !tracker.initCalled {
		t.Error("OnModuleInit not called")
	}
	if !tracker.bootstrapCalled {
		t.Error("OnApplicationBootstrap not called")
	}

	app.Close()

	if !tracker.destroyCalled {
		t.Error("OnModuleDestroy not called")
	}
	if !tracker.beforeShutdown {
		t.Error("BeforeApplicationShutdown not called")
	}
	if !tracker.shutdownCalled {
		t.Error("OnApplicationShutdown not called")
	}
}

func TestLifecycleHooks_Order(t *testing.T) {
	var order []string

	type orderTracker struct{}

	svc := &struct {
		lifecycleTracker
	}{}
	svc.lifecycleTracker = lifecycleTracker{}

	// Use a custom provider that tracks call order
	orderSvc := &orderService{order: &order}

	module := NewModule(ModuleOptions{
		Controllers: []any{func() *lifecycleController { return &lifecycleController{} }},
		Providers:   []any{ProvideValue[*orderService](orderSvc)},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()
	app.Close()

	// Verify init and bootstrap were called
	if len(order) < 2 {
		t.Errorf("expected at least 2 lifecycle events, got %d: %v", len(order), order)
	}
}

type orderService struct {
	order *[]string
}

func (s *orderService) OnModuleInit() error {
	*s.order = append(*s.order, "init")
	return nil
}

func (s *orderService) OnApplicationBootstrap() error {
	*s.order = append(*s.order, "bootstrap")
	return nil
}

func (s *orderService) OnModuleDestroy() error {
	*s.order = append(*s.order, "destroy")
	return nil
}

func (s *orderService) OnApplicationShutdown(signal string) error {
	*s.order = append(*s.order, "shutdown")
	return nil
}

func (s *orderService) BeforeApplicationShutdown(signal string) error {
	*s.order = append(*s.order, "before-shutdown")
	return nil
}
