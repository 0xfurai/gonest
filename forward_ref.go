package gonest

import "reflect"

// ForwardRef wraps a provider constructor to enable circular dependency resolution.
// The inner function is called lazily at resolution time, after all providers are registered.
//
// Usage:
//
//	ForwardRef(func() any { return NewCatService })
type ForwardRef struct {
	Fn func() any
}

// NewForwardRef creates a forward reference. The function should return a constructor
// function that will be used after all providers are registered.
func NewForwardRef(fn func() any) ForwardRef {
	return ForwardRef{Fn: fn}
}

// resolveForwardRef unwraps a ForwardRef if present, returning the inner value.
func resolveForwardRef(v any) any {
	if fwd, ok := v.(ForwardRef); ok {
		return fwd.Fn()
	}
	return v
}

// Optional creates a provider whose dependencies are optional. If a dependency
// cannot be resolved, the zero value is injected rather than returning an error.
func Optional(constructor any) Provider {
	p := Provide(constructor)
	p.optional = true
	return p
}

// ModuleRef provides runtime access to the DI container of a specific module.
// Equivalent to NestJS ModuleRef.
type ModuleRef struct {
	container *Container
}

// NewModuleRef creates a module reference.
func NewModuleRef(container *Container) *ModuleRef {
	return &ModuleRef{container: container}
}

// Get resolves a provider by type from this module's container.
func (ref *ModuleRef) Get(t reflect.Type) (any, error) {
	return ref.container.Resolve(t)
}

// Resolve is a generic helper to resolve a typed instance from the module.
func ModuleRefResolve[T any](ref *ModuleRef) (T, error) {
	return Resolve[T](ref.container)
}

// Has returns true if the module has a provider for the given type.
func (ref *ModuleRef) Has(t reflect.Type) bool {
	return ref.container.Has(t)
}

// ResolveByToken retrieves a provider by string token.
func (ref *ModuleRef) ResolveByToken(token string) (any, error) {
	return ref.container.ResolveByToken(token)
}

// Create resolves a transient instance, bypassing scope rules.
// Useful for creating new instances on demand.
func (ref *ModuleRef) Create(constructor any) (any, error) {
	p := Provide(constructor)
	p.Scope = ScopeTransient
	ref.container.Register(p)
	return ref.container.Resolve(p.Type)
}
