package testing

import (
	"reflect"
	"testing"

	"github.com/0xfurai/gonest"
)

// TestingModule provides a builder for creating test modules with overridden providers.
type TestingModule struct {
	module          *gonest.Module
	overrides       map[reflect.Type]any
	moduleOverrides map[*gonest.Module]*gonest.Module
}

// Test creates a new TestingModule from an existing module.
func Test(module *gonest.Module) *TestingModule {
	return &TestingModule{
		module:          module,
		overrides:       make(map[reflect.Type]any),
		moduleOverrides: make(map[*gonest.Module]*gonest.Module),
	}
}

// OverrideProvider replaces a provider in the module for testing.
// typePtr should be a nil pointer to the type, e.g., (*CatsService)(nil).
// replacement should be a constructor function or value.
func (tm *TestingModule) OverrideProvider(typePtr any, replacement any) *TestingModule {
	t := reflect.TypeOf(typePtr)
	tm.overrides[t] = replacement
	return tm
}

// OverrideModule replaces an entire imported module with a substitute.
// Equivalent to NestJS OverrideModule.
//
// Usage:
//
//	ctm := testing.Test(appModule).
//	    OverrideModule(realDatabaseModule, fakeDatabaseModule).
//	    Compile(t)
func (tm *TestingModule) OverrideModule(original *gonest.Module, replacement *gonest.Module) *TestingModule {
	tm.moduleOverrides[original] = replacement
	return tm
}

// Compile builds the test module and returns a compiled container.
func (tm *TestingModule) Compile(t *testing.T) *CompiledTestModule {
	t.Helper()

	// Create a new module with overridden providers and modules
	opts := tm.module.Options()
	newProviders := make([]any, 0, len(opts.Providers))

	for _, p := range opts.Providers {
		provider := toProvider(p)
		if replacement, ok := tm.overrides[provider.Type]; ok {
			newProviders = append(newProviders, replacement)
		} else {
			newProviders = append(newProviders, p)
		}
	}

	// Apply module overrides to imports
	newImports := make([]*gonest.Module, 0, len(opts.Imports))
	for _, imp := range opts.Imports {
		if replacement, ok := tm.moduleOverrides[imp]; ok {
			newImports = append(newImports, replacement)
		} else {
			newImports = append(newImports, imp)
		}
	}

	testModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     newImports,
		Controllers: opts.Controllers,
		Providers:   newProviders,
		Exports:     opts.Exports,
	})

	app := gonest.Create(testModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("gonest/testing: failed to compile test module: %v", err)
	}

	return &CompiledTestModule{app: app}
}

// CompiledTestModule provides access to the resolved container.
type CompiledTestModule struct {
	app *gonest.Application
}

// Resolve retrieves an instance from the test module's container.
func Resolve[T any](ctm *CompiledTestModule) T {
	t := reflect.TypeOf((*T)(nil)).Elem()
	val, err := ctm.app.Resolve(t)
	if err != nil {
		panic("gonest/testing: failed to resolve " + t.String() + ": " + err.Error())
	}
	return val.(T)
}

// App returns the test application for HTTP testing.
func (ctm *CompiledTestModule) App() *gonest.Application {
	return ctm.app
}

// MockFactory creates mock providers for testing. It generates a zero-value instance
// of the given type that satisfies the interface. For custom mock behavior,
// provide a mock constructor.
//
// Usage:
//
//	mock := testing.MockFactory[*CatsService](func() *MockCatsService {
//	    return &MockCatsService{...}
//	})
//	ctm := testing.Test(module).OverrideProvider((*CatsService)(nil), mock).Compile(t)
type MockProvider[T any] struct {
	factory func() T
}

// MockFactory creates a mock provider from a factory function.
func MockFactory[T any](factory func() T) gonest.Provider {
	return gonest.ProvideValue[T](factory())
}

// AutoMock creates a zero-value mock for the given interface type.
// Useful for interfaces where you don't need any methods to work.
func AutoMock[T any]() gonest.Provider {
	var zero T
	return gonest.ProvideValue[T](zero)
}

func toProvider(p any) gonest.Provider {
	switch v := p.(type) {
	case gonest.Provider:
		return v
	default:
		return gonest.Provide(v)
	}
}
