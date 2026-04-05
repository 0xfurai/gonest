package gonest

import (
	"testing"
)

func TestForwardRef(t *testing.T) {
	called := false
	fwd := NewForwardRef(func() any {
		called = true
		return newGreetingService
	})

	result := resolveForwardRef(fwd)
	if !called {
		t.Error("expected forward ref function to be called")
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestForwardRef_NonForwardRef(t *testing.T) {
	result := resolveForwardRef("hello")
	if result != "hello" {
		t.Errorf("expected passthrough, got %v", result)
	}
}

func TestOptional_Provider(t *testing.T) {
	p := Optional(newGreetingService)
	if !p.optional {
		t.Error("expected optional to be true")
	}
}

func TestModuleRef_Resolve(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	ref, err := Resolve[*ModuleRef](app.GetContainer())
	if err != nil {
		t.Fatalf("resolve ModuleRef: %v", err)
	}

	svc, err := ModuleRefResolve[*greetingService](ref)
	if err != nil {
		t.Fatalf("resolve via ModuleRef: %v", err)
	}
	if svc.greeting != "Hello, World!" {
		t.Errorf("expected greeting, got %q", svc.greeting)
	}
}

func TestConfigurableModuleBuilder(t *testing.T) {
	type MyOpts struct {
		Name string
	}

	builder := NewConfigurableModuleBuilder[MyOpts]()
	mod := builder.Build(MyOpts{Name: "test"}, func(opts MyOpts) ModuleOptions {
		return ModuleOptions{
			Providers: []any{ProvideValue[*MyOpts](&opts)},
			Exports:   []any{(*MyOpts)(nil)},
			Global:    true,
		}
	})

	if !mod.options.Global {
		t.Error("expected module to be global")
	}
	if len(mod.options.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(mod.options.Providers))
	}
}

func TestConfigurableModuleBuilder_SetGlobal(t *testing.T) {
	type MyOpts struct{}

	builder := NewConfigurableModuleBuilder[MyOpts]()
	builder.SetGlobal()

	mod := builder.Build(MyOpts{}, func(opts MyOpts) ModuleOptions {
		return ModuleOptions{}
	})

	if !mod.options.Global {
		t.Error("expected module to be global via builder")
	}
}
