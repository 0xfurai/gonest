package gonest

// Scope defines the lifetime of a provider instance.
type Scope int

const (
	// ScopeSingleton creates one instance shared across the entire application.
	ScopeSingleton Scope = iota
	// ScopeRequest creates a new instance for each incoming request.
	ScopeRequest
	// ScopeTransient creates a new instance every time it is injected.
	ScopeTransient
)

// String returns the string representation of a Scope.
func (s Scope) String() string {
	switch s {
	case ScopeSingleton:
		return "singleton"
	case ScopeRequest:
		return "request"
	case ScopeTransient:
		return "transient"
	default:
		return "unknown"
	}
}
