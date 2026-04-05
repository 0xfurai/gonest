package gonest

import (
	"fmt"
	"reflect"
	"sync"
)

// LazyModuleLoader loads modules on-demand at runtime, after bootstrap.
// Equivalent to NestJS LazyModuleLoader.
//
// Usage:
//
//	loader := app.GetLazyModuleLoader()
//	mod, err := loader.Load(func() *Module {
//	    return NewModule(ModuleOptions{
//	        Providers: []any{NewExpensiveService},
//	        Exports:   []any{(*ExpensiveService)(nil)},
//	    })
//	})
//	svc, _ := mod.Get(reflect.TypeOf((*ExpensiveService)(nil)))
type LazyModuleLoader struct {
	app     *Application
	ctx     *ApplicationContext
	mu      sync.Mutex
	loaded  []*LazyLoadedModule
}

// LazyLoadedModule wraps a lazily loaded module with its resolved container.
type LazyLoadedModule struct {
	module    *Module
	container *Container
}

// Get resolves a provider by type from the lazy-loaded module.
func (lm *LazyLoadedModule) Get(t reflect.Type) (any, error) {
	return lm.container.Resolve(t)
}

// Resolve is a generic helper to resolve a typed instance from the lazy module.
func LazyModuleResolve[T any](lm *LazyLoadedModule) (T, error) {
	return Resolve[T](lm.container)
}

// NewLazyModuleLoader creates a lazy module loader for an HTTP application.
func NewLazyModuleLoader(app *Application) *LazyModuleLoader {
	return &LazyModuleLoader{app: app}
}

// NewLazyModuleLoaderFromContext creates a lazy module loader for an application context.
func NewLazyModuleLoaderFromContext(ctx *ApplicationContext) *LazyModuleLoader {
	return &LazyModuleLoader{ctx: ctx}
}

// Load lazily loads a module at runtime. The factory function is called to create
// the module, which is then compiled against the existing application container.
// The module's providers are resolved and lifecycle hooks are called.
func (l *LazyModuleLoader) Load(factory func() *Module) (*LazyLoadedModule, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	mod := factory()
	if mod == nil {
		return nil, fmt.Errorf("gonest: lazy module factory returned nil")
	}

	var parentContainer *Container
	var logger Logger
	var reflector *Reflector

	if l.app != nil {
		parentContainer = l.app.module.container
		logger = l.app.logger
		reflector = l.app.reflector
	} else if l.ctx != nil {
		parentContainer = l.ctx.module.container
		logger = l.ctx.logger
		reflector = l.ctx.reflector
	} else {
		return nil, fmt.Errorf("gonest: lazy module loader has no application context")
	}

	// Compile the module against the parent container
	if err := mod.compile(parentContainer, logger, reflector); err != nil {
		return nil, fmt.Errorf("gonest: lazy module compilation failed: %w", err)
	}

	// Run OnApplicationBootstrap hooks for the newly loaded module
	instances, err := mod.container.ResolveAll()
	if err != nil {
		return nil, fmt.Errorf("gonest: lazy module resolution failed: %w", err)
	}
	for _, inst := range instances {
		if hook, ok := inst.(OnApplicationBootstrap); ok {
			if err := hook.OnApplicationBootstrap(); err != nil {
				return nil, err
			}
		}
	}

	// Update discovery service and graph inspector
	if l.app != nil {
		allMods := append(l.app.module.allModules(), mod)
		l.app.discovery.SetModules(allMods)
		l.app.graphInspector.SetModules(allMods)
	}

	lm := &LazyLoadedModule{
		module:    mod,
		container: mod.container,
	}
	l.loaded = append(l.loaded, lm)

	logger.Log("Lazy loaded module with %d providers", len(instances))
	return lm, nil
}

// GetLoadedModules returns all lazily loaded modules.
func (l *LazyModuleLoader) GetLoadedModules() []*LazyLoadedModule {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]*LazyLoadedModule{}, l.loaded...)
}
