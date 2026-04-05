package gonest

import "reflect"

// ModuleOptions defines the configuration for a module.
type ModuleOptions struct {
	// Imports lists other modules whose exported providers are available in this module.
	Imports []*Module
	// Controllers lists controller constructor functions.
	// Each must be a function returning a type that implements Controller.
	Controllers []any
	// Providers lists provider constructor functions, values, or Provider structs.
	Providers []any
	// Exports lists types (as nil pointers) or Provider structs that this module
	// makes available to importing modules.
	Exports []any
	// Global makes this module's exports available to all modules.
	Global bool
}

// Module represents a cohesive unit of the application.
type Module struct {
	options     ModuleOptions
	container   *Container
	controllers []Controller
	configured  bool
	configurer  MiddlewareConfigurer
}

// NewModule creates a new module from the given options.
func NewModule(opts ModuleOptions) *Module {
	return &Module{options: opts}
}

// DynamicModule supports runtime module configuration via ForRoot/ForFeature patterns.
// Use NewDynamicModule to create one from a base module with additional providers.
type DynamicModule struct {
	Module    *Module
	Providers []any
	Exports   []any
	Global    bool
}

// NewDynamicModule creates a module with runtime-configured providers.
// This enables the ForRoot/ForFeature pattern used in NestJS.
//
//	var ConfigModule = gonest.NewDynamicModule(gonest.DynamicModule{
//	    Providers: []any{gonest.ProvideValue[*Options](opts)},
//	    Exports:   []any{(*Options)(nil)},
//	    Global:    true,
//	})
func NewDynamicModule(dm DynamicModule) *Module {
	return NewModule(ModuleOptions{
		Providers: dm.Providers,
		Exports:   dm.Exports,
		Global:    dm.Global,
	})
}

// ForRoot is a convention helper for creating a configured module.
// factoryFn receives the options and returns a *Module.
func ForRoot[T any](opts T, factoryFn func(T) *Module) *Module {
	return factoryFn(opts)
}

// ForFeature is a convention helper for feature-scoped module configuration.
func ForFeature[T any](opts T, factoryFn func(T) *Module) *Module {
	return factoryFn(opts)
}

// Options returns the module's options (used by the testing package).
func (m *Module) Options() ModuleOptions {
	return m.options
}

// compile resolves all providers and controllers within this module.
func (m *Module) compile(parentContainer *Container, logger Logger, reflector *Reflector) error {
	if m.configured {
		return nil
	}
	m.configured = true

	if parentContainer != nil {
		m.container = NewChildContainer(parentContainer)
	} else {
		m.container = NewContainer(logger)
	}

	// Register the reflector so guards/interceptors can inject it
	m.container.RegisterInstance(reflect.TypeOf((*Reflector)(nil)), reflector)

	// Register the logger
	m.container.RegisterInstance(reflect.TypeOf((*DefaultLogger)(nil)), logger)
	// Also register under Logger interface
	loggerIfaceType := reflect.TypeOf((*Logger)(nil)).Elem()
	m.container.RegisterInstance(loggerIfaceType, logger)

	// Compile imported modules first.
	// Each child module gets this module's container as parent, so it can
	// resolve types exported by previously-compiled sibling modules.
	for _, imp := range m.options.Imports {
		if err := imp.compile(m.container, logger, reflector); err != nil {
			return err
		}
		// Import exported providers into this module's container
		for _, exp := range imp.options.Exports {
			exportType := resolveExportType(exp)
			if exportType != nil && imp.container.Has(exportType) {
				instance, err := imp.container.Resolve(exportType)
				if err != nil {
					return err
				}
				m.container.RegisterInstance(exportType, instance)
			}
		}
		// Global modules export to the root by propagating up the parent chain
		if imp.options.Global && parentContainer != nil {
			for _, exp := range imp.options.Exports {
				exportType := resolveExportType(exp)
				if exportType != nil && imp.container.Has(exportType) {
					instance, err := imp.container.Resolve(exportType)
					if err != nil {
						return err
					}
					parentContainer.RegisterInstance(exportType, instance)
				}
			}
		}
	}

	// Register ModuleRef for this module
	moduleRef := NewModuleRef(m.container)
	m.container.RegisterInstance(reflect.TypeOf((*ModuleRef)(nil)), moduleRef)

	// Register providers (resolving forward refs)
	for _, p := range m.options.Providers {
		resolved := resolveForwardRef(p)
		provider := toProvider(resolved)
		m.container.Register(provider)
	}

	// Resolve and register controllers
	for _, ctrlFactory := range m.options.Controllers {
		provider := toProvider(ctrlFactory)
		m.container.Register(provider)

		instance, err := m.container.Resolve(provider.Type)
		if err != nil {
			return err
		}
		ctrl, ok := instance.(Controller)
		if !ok {
			return NewInternalServerError("controller does not implement gonest.Controller: " + provider.Type.String())
		}
		m.controllers = append(m.controllers, ctrl)
	}

	// Run lifecycle hooks: OnModuleInit
	allProviders, err := m.container.ResolveAll()
	if err != nil {
		return err
	}
	for _, p := range allProviders {
		if init, ok := p.(OnModuleInit); ok {
			if err := init.OnModuleInit(); err != nil {
				return err
			}
		}
	}

	return nil
}

// destroy runs OnModuleDestroy for all providers.
func (m *Module) destroy() error {
	if m.container == nil {
		return nil
	}
	allProviders, err := m.container.ResolveAll()
	if err != nil {
		return err
	}
	for _, p := range allProviders {
		if d, ok := p.(OnModuleDestroy); ok {
			if err := d.OnModuleDestroy(); err != nil {
				return err
			}
		}
	}
	// Recurse into imported modules
	for _, imp := range m.options.Imports {
		if err := imp.destroy(); err != nil {
			return err
		}
	}
	return nil
}

// toProvider normalizes various provider formats into a Provider struct.
func toProvider(p any) Provider {
	switch v := p.(type) {
	case Provider:
		return v
	default:
		// Assume it's a constructor function
		return Provide(v)
	}
}

// resolveExportType extracts the reflect.Type from an export declaration.
func resolveExportType(exp any) reflect.Type {
	switch v := exp.(type) {
	case Provider:
		if v.InterfaceType != nil {
			return v.InterfaceType
		}
		return v.Type
	default:
		// Assume it's a nil pointer like (*CatsService)(nil)
		t := reflect.TypeOf(v)
		if t == nil {
			return nil
		}
		if t.Kind() == reflect.Ptr {
			return t
		}
		return t
	}
}

// allControllers collects all controllers from this module and its imports.
func (m *Module) allControllers() []Controller {
	var ctrls []Controller
	for _, imp := range m.options.Imports {
		ctrls = append(ctrls, imp.allControllers()...)
	}
	ctrls = append(ctrls, m.controllers...)
	return ctrls
}

// allModules collects this module and all imported modules.
func (m *Module) allModules() []*Module {
	var mods []*Module
	for _, imp := range m.options.Imports {
		mods = append(mods, imp.allModules()...)
	}
	mods = append(mods, m)
	return mods
}
