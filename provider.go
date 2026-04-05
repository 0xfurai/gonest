package gonest

import "reflect"

// ProviderType classifies how a provider is created.
type ProviderType int

const (
	ProviderTypeConstructor ProviderType = iota
	ProviderTypeValue
	ProviderTypeFactory
)

// Provider holds the metadata needed for the DI container to create an instance.
type Provider struct {
	// Type is the reflect.Type this provider produces (the return type).
	Type reflect.Type
	// Constructor is a function whose parameters are resolved via DI.
	// Signature: func(dep1 T1, dep2 T2, ...) ReturnType
	Constructor any
	// Value is a pre-built instance (for value providers).
	Value any
	// ProviderType indicates how this provider should be resolved.
	ProviderType ProviderType
	// Scope controls the provider's lifetime.
	Scope Scope
	// Token is an optional string token for non-type-based injection.
	Token string
	// InterfaceType, if set, means this provider satisfies an interface binding.
	InterfaceType reflect.Type
	// optional marks this provider's dependencies as optional.
	optional bool
}

// Provide creates a constructor-based provider. The constructor's return type
// is used as the provider's type key. Dependencies are resolved from the container.
//
// Usage: Provide(NewCatsService)
func Provide(constructor any) Provider {
	ct := reflect.TypeOf(constructor)
	if ct.Kind() != reflect.Func {
		panic("gonest.Provide: argument must be a function")
	}
	if ct.NumOut() < 1 {
		panic("gonest.Provide: constructor must return at least one value")
	}
	return Provider{
		Type:         ct.Out(0),
		Constructor:  constructor,
		ProviderType: ProviderTypeConstructor,
		Scope:        ScopeSingleton,
	}
}

// ProvideWithScope creates a constructor-based provider with a specific scope.
func ProvideWithScope(constructor any, scope Scope) Provider {
	p := Provide(constructor)
	p.Scope = scope
	return p
}

// ProvideValue creates a value provider from an existing instance.
//
// Usage: ProvideValue[Logger](myLogger)
func ProvideValue[T any](value T) Provider {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return Provider{
		Type:         t,
		Value:        value,
		ProviderType: ProviderTypeValue,
		Scope:        ScopeSingleton,
	}
}

// ProvideFactory creates a factory provider. The factory function's
// parameters are resolved from the DI container.
func ProvideFactory[T any](factory any) Provider {
	t := reflect.TypeOf((*T)(nil)).Elem()
	ft := reflect.TypeOf(factory)
	if ft.Kind() != reflect.Func {
		panic("gonest.ProvideFactory: argument must be a function")
	}
	return Provider{
		Type:         t,
		Constructor:  factory,
		ProviderType: ProviderTypeFactory,
		Scope:        ScopeSingleton,
	}
}

// Bind creates a provider that binds an interface to a concrete implementation.
// The constructor builds the concrete type, which is stored under the interface type.
//
// Usage: Bind[Repository](NewMemoryRepository)
func Bind[Iface any](constructor any) Provider {
	ifaceType := reflect.TypeOf((*Iface)(nil)).Elem()
	ct := reflect.TypeOf(constructor)
	if ct.Kind() != reflect.Func {
		panic("gonest.Bind: argument must be a function")
	}
	return Provider{
		Type:          ct.Out(0),
		InterfaceType: ifaceType,
		Constructor:   constructor,
		ProviderType:  ProviderTypeConstructor,
		Scope:         ScopeSingleton,
	}
}

// ProvideToken creates a provider accessible by a string token.
func ProvideToken(token string, constructor any) Provider {
	p := Provide(constructor)
	p.Token = token
	return p
}

// ProvideTokenValue creates a value provider accessible by a string token.
func ProvideTokenValue(token string, value any) Provider {
	return Provider{
		Type:         reflect.TypeOf(value),
		Value:        value,
		ProviderType: ProviderTypeValue,
		Scope:        ScopeSingleton,
		Token:        token,
	}
}

// tokenType is used internally to store token-based providers.
type tokenType struct {
	token string
}

// InjectToken is used as a constructor parameter type to request a token-based provider.
type InjectToken struct {
	Token string
}
