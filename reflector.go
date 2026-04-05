package gonest

import "sync"

// Reflector stores and retrieves metadata associated with route handlers.
// This is the Go equivalent of NestJS's Reflector, used by guards and
// interceptors to read metadata set on routes via SetMetadata.
type Reflector struct {
	mu   sync.RWMutex
	data map[any]map[string]any // key -> metadata key -> value
}

func NewReflector() *Reflector {
	return &Reflector{
		data: make(map[any]map[string]any),
	}
}

// Set stores a metadata value for a given target (typically a route handler).
func (r *Reflector) Set(target any, key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.data[target]
	if !ok {
		m = make(map[string]any)
		r.data[target] = m
	}
	m[key] = value
}

// Get retrieves a metadata value for a given target.
func (r *Reflector) Get(target any, key string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.data[target]
	if !ok {
		return nil, false
	}
	v, ok := m[key]
	return v, ok
}

// GetAll returns all metadata for a given target.
func (r *Reflector) GetAll(target any) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m, ok := r.data[target]
	if !ok {
		return nil
	}
	cp := make(map[string]any, len(m))
	for k, v := range m {
		cp[k] = v
	}
	return cp
}

// GetMetadata is a generic helper to retrieve typed metadata from an ExecutionContext.
func GetMetadata[T any](ctx ExecutionContext, key string) (T, bool) {
	val, ok := ctx.GetMetadata(key)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := val.(T)
	return typed, ok
}
