package gonest

import "testing"

func TestConfigurableModuleBuilder_BuildAsync(t *testing.T) {
	type DBOpts struct {
		DSN string
	}

	builder := NewConfigurableModuleBuilder[DBOpts]()

	mod := builder.BuildAsync(
		AsyncModuleOptions[DBOpts]{
			Factory: func() (DBOpts, error) {
				return DBOpts{DSN: "postgres://localhost/test"}, nil
			},
			Global: true,
		},
		func(opts DBOpts) ModuleOptions {
			return ModuleOptions{
				Providers: []any{ProvideValue[*DBOpts](&opts)},
				Exports:   []any{(*DBOpts)(nil)},
			}
		},
	)

	if !mod.options.Global {
		t.Error("expected async module to be global")
	}
	if len(mod.options.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(mod.options.Providers))
	}
}

func TestConfigurableModuleBuilder_AsyncWithImports(t *testing.T) {
	type Opts struct{ Key string }

	depModule := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
		Exports:   []any{(*greetingService)(nil)},
	})

	builder := NewConfigurableModuleBuilder[Opts]()
	mod := builder.BuildAsync(
		AsyncModuleOptions[Opts]{
			Imports: []*Module{depModule},
			Factory: func() (Opts, error) {
				return Opts{Key: "abc"}, nil
			},
		},
		func(opts Opts) ModuleOptions {
			return ModuleOptions{
				Providers: []any{ProvideValue[*Opts](&opts)},
			}
		},
	)

	// Imports from AsyncModuleOptions should be merged
	if len(mod.options.Imports) != 1 {
		t.Errorf("expected 1 import, got %d", len(mod.options.Imports))
	}
}
