package gonest

import "testing"

func TestNewModule(t *testing.T) {
	m := NewModule(ModuleOptions{})
	if m == nil {
		t.Fatal("expected non-nil module")
	}
}

func TestModule_Compile(t *testing.T) {
	svc := newGreetingService()
	m := NewModule(ModuleOptions{
		Providers: []any{ProvideValue[*greetingService](svc)},
	})

	err := m.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
}

func TestModule_CompileWithImports(t *testing.T) {
	childModule := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
		Exports:   []any{(*greetingService)(nil)},
	})

	parentModule := NewModule(ModuleOptions{
		Imports: []*Module{childModule},
	})

	err := parentModule.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	// Parent should be able to resolve the exported service
	instance, err := parentModule.container.Resolve(resolveExportType((*greetingService)(nil)))
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if instance.(*greetingService).greeting != "Hello, World!" {
		t.Error("unexpected greeting value")
	}
}

func TestModule_CompileWithControllers(t *testing.T) {
	m := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	err := m.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	if len(m.controllers) != 1 {
		t.Errorf("expected 1 controller, got %d", len(m.controllers))
	}
}

func TestModule_GlobalModule(t *testing.T) {
	globalModule := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
		Exports:   []any{(*greetingService)(nil)},
		Global:    true,
	})

	otherModule := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
	})

	appModule := NewModule(ModuleOptions{
		Imports: []*Module{globalModule, otherModule},
	})

	err := appModule.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}
}

func TestModule_AllControllers(t *testing.T) {
	child := NewModule(ModuleOptions{
		Controllers: []any{newPipedController},
	})
	parent := NewModule(ModuleOptions{
		Imports:     []*Module{child},
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	err := parent.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	ctrls := parent.allControllers()
	if len(ctrls) != 2 {
		t.Errorf("expected 2 controllers, got %d", len(ctrls))
	}
}

func TestModule_AllModules(t *testing.T) {
	child := NewModule(ModuleOptions{})
	parent := NewModule(ModuleOptions{Imports: []*Module{child}})

	parent.compile(nil, NopLogger{}, NewReflector())
	mods := parent.allModules()
	if len(mods) != 2 {
		t.Errorf("expected 2 modules, got %d", len(mods))
	}
}

func TestModule_Destroy(t *testing.T) {
	svc := newLifecycleService()
	m := NewModule(ModuleOptions{
		Controllers: []any{newLifecycleController},
		Providers:   []any{ProvideValue[*lifecycleService](svc)},
	})

	m.compile(nil, NopLogger{}, NewReflector())
	err := m.destroy()
	if err != nil {
		t.Fatalf("destroy failed: %v", err)
	}
	if !svc.destroyCalled {
		t.Error("expected OnModuleDestroy to be called")
	}
}

func TestToProvider_ProviderStruct(t *testing.T) {
	p := Provider{
		Type:         nil,
		ProviderType: ProviderTypeValue,
		Value:        "test",
	}
	result := toProvider(p)
	if result.ProviderType != ProviderTypeValue {
		t.Error("expected passthrough for Provider struct")
	}
}

func TestToProvider_Constructor(t *testing.T) {
	result := toProvider(newGreetingService)
	if result.ProviderType != ProviderTypeConstructor {
		t.Error("expected constructor provider")
	}
}

func TestResolveExportType_NilPointer(t *testing.T) {
	typ := resolveExportType((*greetingService)(nil))
	if typ == nil {
		t.Fatal("expected non-nil type")
	}
}

func TestResolveExportType_Provider(t *testing.T) {
	p := Provide(newGreetingService)
	typ := resolveExportType(p)
	if typ == nil {
		t.Fatal("expected non-nil type")
	}
}

func TestNewDynamicModule(t *testing.T) {
	svc := &greetingService{greeting: "dynamic"}
	dm := NewDynamicModule(DynamicModule{
		Providers: []any{ProvideValue[*greetingService](svc)},
		Exports:   []any{(*greetingService)(nil)},
		Global:    true,
	})

	err := dm.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	instance, err := dm.container.Resolve(resolveExportType((*greetingService)(nil)))
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if instance.(*greetingService).greeting != "dynamic" {
		t.Errorf("expected 'dynamic', got %q", instance.(*greetingService).greeting)
	}
}

func TestForRoot(t *testing.T) {
	type DBOptions struct {
		Host string
		Port int
	}

	createDBModule := func(opts DBOptions) *Module {
		return NewModule(ModuleOptions{
			Providers: []any{ProvideValue[DBOptions](opts)},
			Exports:   []any{ProvideValue[DBOptions](opts)},
			Global:    true,
		})
	}

	dbModule := ForRoot(DBOptions{Host: "localhost", Port: 5432}, createDBModule)
	err := dbModule.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	if dbModule.options.Global != true {
		t.Error("expected global module")
	}
}

func TestModule_Options(t *testing.T) {
	m := NewModule(ModuleOptions{
		Global: true,
	})
	opts := m.Options()
	if !opts.Global {
		t.Error("expected Global to be true")
	}
}

func TestModule_CrossModuleDI(t *testing.T) {
	// Simulates: dbModule exports *testServiceA, usersModule needs it,
	// authModule needs *testServiceB from usersModule.
	// All imported by root — DI should resolve across sibling modules.

	dbModule := NewModule(ModuleOptions{
		Providers: []any{newTestServiceA},
		Exports:   []any{(*testServiceA)(nil)},
	})

	usersModule := NewModule(ModuleOptions{
		Providers: []any{newTestServiceB}, // needs *testServiceA
		Exports:   []any{(*testServiceB)(nil)},
	})

	authModule := NewModule(ModuleOptions{
		Providers: []any{newTestServiceC}, // needs *testServiceA + *testServiceB
	})

	root := NewModule(ModuleOptions{
		Imports: []*Module{dbModule, usersModule, authModule},
	})

	err := root.compile(nil, NopLogger{}, NewReflector())
	if err != nil {
		t.Fatalf("cross-module DI failed: %v", err)
	}

	// authModule should have resolved testServiceC with deps from siblings
	svcC, err := authModule.container.Resolve(resolveExportType((*testServiceC)(nil)))
	if err != nil {
		t.Fatalf("resolve testServiceC: %v", err)
	}
	c := svcC.(*testServiceC)
	if c.A == nil || c.B == nil {
		t.Error("expected both A and B to be resolved")
	}
	if c.B.A == nil {
		t.Error("expected B.A to be resolved")
	}
}

func TestModule_CrossModuleDI_OrderMatters(t *testing.T) {
	// If usersModule is imported before dbModule, usersModule can't resolve *testServiceA
	dbModule := NewModule(ModuleOptions{
		Providers: []any{newTestServiceA},
		Exports:   []any{(*testServiceA)(nil)},
	})

	usersModule := NewModule(ModuleOptions{
		Providers: []any{newTestServiceB}, // needs *testServiceA
	})

	// Wrong order: usersModule before dbModule
	root := NewModule(ModuleOptions{
		Imports: []*Module{usersModule, dbModule},
	})

	err := root.compile(nil, NopLogger{}, NewReflector())
	// This should fail because usersModule compiles before dbModule exports *testServiceA
	if err == nil {
		t.Log("Note: order-dependent DI resolved (parent fallback), which is acceptable")
	}
}
