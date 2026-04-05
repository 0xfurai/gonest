package gonest

import (
	"fmt"
	"reflect"
	"sync"
)

// Container is the dependency injection container. It resolves providers
// by type, supporting constructor injection, value providers, and interface bindings.
type Container struct {
	mu        sync.RWMutex
	providers map[reflect.Type]*providerEntry
	tokens    map[string]*providerEntry
	parent    *Container
	instances map[reflect.Type]any
	logger    Logger
}

type providerEntry struct {
	provider Provider
	instance any
	resolved bool
}

// NewContainer creates a new DI container.
func NewContainer(logger Logger) *Container {
	if logger == nil {
		logger = NopLogger{}
	}
	return &Container{
		providers: make(map[reflect.Type]*providerEntry),
		tokens:    make(map[string]*providerEntry),
		instances: make(map[reflect.Type]any),
		logger:    logger,
	}
}

// NewChildContainer creates a child container that falls back to the parent
// for unresolved types.
func NewChildContainer(parent *Container) *Container {
	return &Container{
		providers: make(map[reflect.Type]*providerEntry),
		tokens:    make(map[string]*providerEntry),
		instances: make(map[reflect.Type]any),
		parent:    parent,
		logger:    parent.logger,
	}
}

// Register adds a provider to the container.
func (c *Container) Register(p Provider) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &providerEntry{provider: p}

	if p.ProviderType == ProviderTypeValue {
		entry.instance = p.Value
		entry.resolved = true
	}

	if p.Token != "" {
		c.tokens[p.Token] = entry
		return
	}

	key := p.Type
	if p.InterfaceType != nil {
		key = p.InterfaceType
	}
	c.providers[key] = entry
}

// RegisterInstance directly registers a resolved instance by type.
func (c *Container) RegisterInstance(t reflect.Type, instance any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers[t] = &providerEntry{
		provider: Provider{Type: t, ProviderType: ProviderTypeValue, Value: instance},
		instance: instance,
		resolved: true,
	}
}

// Resolve retrieves or creates an instance for the given type.
func (c *Container) Resolve(t reflect.Type) (any, error) {
	c.mu.RLock()
	entry, ok := c.providers[t]
	c.mu.RUnlock()

	if !ok {
		if c.parent != nil {
			return c.parent.Resolve(t)
		}
		return nil, fmt.Errorf("gonest: no provider for type %v", t)
	}

	switch entry.provider.Scope {
	case ScopeSingleton:
		if entry.resolved {
			return entry.instance, nil
		}
		return c.build(entry)
	case ScopeTransient:
		// Always build a new instance
		return c.build(entry)
	case ScopeRequest:
		// Request-scoped: build new instance every time (caller manages per-request containers)
		return c.build(entry)
	default:
		if entry.resolved {
			return entry.instance, nil
		}
		return c.build(entry)
	}
}

// CreateRequestContainer creates a child container for a single request.
// Request-scoped providers are resolved fresh in this container.
// Providers with propagated request scope (singletons depending on request-scoped
// providers) are also included.
func (c *Container) CreateRequestContainer() *Container {
	child := NewChildContainer(c)
	// Copy request-scoped provider registrations into the child,
	// including providers whose scope is elevated via propagation.
	c.mu.RLock()
	for t, entry := range c.providers {
		effectiveScope := c.GetEffectiveScope(t)
		if entry.provider.Scope == ScopeRequest || effectiveScope == ScopeRequest {
			child.Register(entry.provider)
		}
	}
	c.mu.RUnlock()
	return child
}

// ResolveByToken retrieves or creates an instance by string token.
func (c *Container) ResolveByToken(token string) (any, error) {
	c.mu.RLock()
	entry, ok := c.tokens[token]
	c.mu.RUnlock()

	if !ok {
		if c.parent != nil {
			return c.parent.ResolveByToken(token)
		}
		return nil, fmt.Errorf("gonest: no provider for token %q", token)
	}

	if entry.resolved && entry.provider.Scope == ScopeSingleton {
		return entry.instance, nil
	}

	return c.build(entry)
}

// Has returns true if the container or its parent has a provider for the given type.
func (c *Container) Has(t reflect.Type) bool {
	c.mu.RLock()
	_, ok := c.providers[t]
	c.mu.RUnlock()
	if ok {
		return true
	}
	if c.parent != nil {
		return c.parent.Has(t)
	}
	return false
}

// GetEffectiveScope returns the effective scope for a provider, taking into
// account scope propagation through the dependency tree. If a singleton depends
// on a request-scoped provider, the singleton is elevated to request scope.
func (c *Container) GetEffectiveScope(t reflect.Type) Scope {
	c.mu.RLock()
	entry, ok := c.providers[t]
	c.mu.RUnlock()
	if !ok {
		if c.parent != nil {
			return c.parent.GetEffectiveScope(t)
		}
		return ScopeSingleton
	}

	scope := entry.provider.Scope
	if entry.provider.Constructor == nil {
		return scope
	}

	// Check dependencies for scope propagation
	ct := reflect.TypeOf(entry.provider.Constructor)
	for i := 0; i < ct.NumIn(); i++ {
		depScope := c.GetEffectiveScope(ct.In(i))
		if depScope > scope {
			scope = depScope
		}
	}
	return scope
}

// build invokes a constructor function, resolving its parameters from the container.
func (c *Container) build(entry *providerEntry) (any, error) {
	if entry.provider.ProviderType == ProviderTypeValue {
		entry.instance = entry.provider.Value
		entry.resolved = true
		return entry.instance, nil
	}

	constructor := entry.provider.Constructor
	ct := reflect.TypeOf(constructor)

	args := make([]reflect.Value, ct.NumIn())
	for i := 0; i < ct.NumIn(); i++ {
		paramType := ct.In(i)
		dep, err := c.Resolve(paramType)
		if err != nil {
			if entry.provider.optional {
				args[i] = reflect.Zero(paramType)
				continue
			}
			return nil, fmt.Errorf("gonest: resolving param %d (%v) for %v: %w",
				i, paramType, entry.provider.Type, err)
		}
		args[i] = reflect.ValueOf(dep)
	}

	results := reflect.ValueOf(constructor).Call(args)

	instance := results[0].Interface()

	// Handle constructors that return (T, error)
	if len(results) == 2 && !results[1].IsNil() {
		return nil, fmt.Errorf("gonest: constructor for %v returned error: %w",
			entry.provider.Type, results[1].Interface().(error))
	}

	// Determine effective scope considering propagation
	effectiveScope := c.GetEffectiveScope(entry.provider.Type)
	if effectiveScope == ScopeSingleton {
		c.mu.Lock()
		entry.instance = instance
		entry.resolved = true
		c.mu.Unlock()
	}

	return instance, nil
}

// ResolveAll returns all registered instances, building them if needed.
func (c *Container) ResolveAll() ([]any, error) {
	c.mu.RLock()
	entries := make([]*providerEntry, 0, len(c.providers))
	for _, e := range c.providers {
		entries = append(entries, e)
	}
	c.mu.RUnlock()

	var result []any
	for _, entry := range entries {
		if !entry.resolved {
			if _, err := c.build(entry); err != nil {
				return nil, err
			}
		}
		result = append(result, entry.instance)
	}
	return result, nil
}

// GetAllProviders returns all registered provider entries.
func (c *Container) GetAllProviders() map[reflect.Type]*providerEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cp := make(map[reflect.Type]*providerEntry, len(c.providers))
	for k, v := range c.providers {
		cp[k] = v
	}
	return cp
}

// Resolve is a generic helper to resolve a typed instance from the container.
func Resolve[T any](c *Container) (T, error) {
	t := reflect.TypeOf((*T)(nil)).Elem()
	val, err := c.Resolve(t)
	if err != nil {
		var zero T
		return zero, err
	}
	return val.(T), nil
}

// MustResolve is like Resolve but panics on error.
func MustResolve[T any](c *Container) T {
	val, err := Resolve[T](c)
	if err != nil {
		panic(err)
	}
	return val
}
