package gonest

// ConfigurableModuleBuilder provides a utility for building dynamic modules
// that accept configuration. Equivalent to NestJS ConfigurableModuleBuilder.
//
// Usage:
//
//	type MyModuleOptions struct { ApiKey string }
//
//	var builder = gonest.NewConfigurableModuleBuilder[MyModuleOptions]()
//
//	func NewMyModule(opts MyModuleOptions) *gonest.Module {
//	    return builder.Build(opts, func(opts MyModuleOptions) gonest.ModuleOptions {
//	        return gonest.ModuleOptions{
//	            Providers: []any{gonest.ProvideValue[*MyModuleOptions](&opts)},
//	            Exports:   []any{(*MyModuleOptions)(nil)},
//	        }
//	    })
//	}
//
//	// Async configuration
//	func NewMyModuleAsync(asyncOpts gonest.AsyncModuleOptions[MyModuleOptions]) *gonest.Module {
//	    return builder.BuildAsync(asyncOpts, func(opts MyModuleOptions) gonest.ModuleOptions {
//	        return gonest.ModuleOptions{
//	            Providers: []any{gonest.ProvideValue[*MyModuleOptions](&opts)},
//	            Exports:   []any{(*MyModuleOptions)(nil)},
//	        }
//	    })
//	}
type ConfigurableModuleBuilder[T any] struct {
	global bool
}

// NewConfigurableModuleBuilder creates a new configurable module builder.
func NewConfigurableModuleBuilder[T any]() *ConfigurableModuleBuilder[T] {
	return &ConfigurableModuleBuilder[T]{}
}

// SetGlobal makes all modules built by this builder global.
func (b *ConfigurableModuleBuilder[T]) SetGlobal() *ConfigurableModuleBuilder[T] {
	b.global = true
	return b
}

// Build creates a module with synchronous configuration.
func (b *ConfigurableModuleBuilder[T]) Build(opts T, configure func(T) ModuleOptions) *Module {
	modOpts := configure(opts)
	modOpts.Global = b.global || modOpts.Global
	return NewModule(modOpts)
}

// AsyncModuleOptions provides async configuration for a module.
// Imports are resolved first, then the factory is called with dependencies.
type AsyncModuleOptions[T any] struct {
	// Imports lists modules whose exports are available to the factory.
	Imports []*Module
	// Factory creates the options using injected dependencies.
	Factory func() (T, error)
	// Global makes this module global.
	Global bool
}

// BuildAsync creates a module with async (factory-based) configuration.
func (b *ConfigurableModuleBuilder[T]) BuildAsync(
	asyncOpts AsyncModuleOptions[T],
	configure func(T) ModuleOptions,
) *Module {
	opts, err := asyncOpts.Factory()
	if err != nil {
		panic("gonest: ConfigurableModuleBuilder async factory error: " + err.Error())
	}

	modOpts := configure(opts)
	modOpts.Imports = append(asyncOpts.Imports, modOpts.Imports...)
	modOpts.Global = b.global || asyncOpts.Global || modOpts.Global
	return NewModule(modOpts)
}
