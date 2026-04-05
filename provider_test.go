package gonest

import (
	"reflect"
	"testing"
)

func TestProvide_Constructor(t *testing.T) {
	p := Provide(newTestServiceA)
	if p.ProviderType != ProviderTypeConstructor {
		t.Errorf("expected ProviderTypeConstructor, got %d", p.ProviderType)
	}
	if p.Type != reflect.TypeOf((*testServiceA)(nil)) {
		t.Errorf("unexpected type: %v", p.Type)
	}
	if p.Scope != ScopeSingleton {
		t.Errorf("expected ScopeSingleton, got %d", p.Scope)
	}
}

func TestProvide_PanicOnNonFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	Provide("not a function")
}

func TestProvideWithScope(t *testing.T) {
	p := ProvideWithScope(newTestServiceA, ScopeTransient)
	if p.Scope != ScopeTransient {
		t.Errorf("expected ScopeTransient, got %d", p.Scope)
	}
}

func TestProvideValue(t *testing.T) {
	svc := &testServiceA{Value: "val"}
	p := ProvideValue[*testServiceA](svc)
	if p.ProviderType != ProviderTypeValue {
		t.Errorf("expected ProviderTypeValue, got %d", p.ProviderType)
	}
	if p.Value != svc {
		t.Error("expected same value")
	}
}

func TestBind(t *testing.T) {
	p := Bind[testRepository](newTestMemoryRepo)
	if p.InterfaceType != reflect.TypeOf((*testRepository)(nil)).Elem() {
		t.Errorf("unexpected interface type: %v", p.InterfaceType)
	}
}

func TestBind_PanicOnNonFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	Bind[testRepository]("not a function")
}

func TestProvideToken(t *testing.T) {
	p := ProvideToken("MY_TOKEN", newTestServiceA)
	if p.Token != "MY_TOKEN" {
		t.Errorf("expected 'MY_TOKEN', got %q", p.Token)
	}
}

func TestProvideTokenValue(t *testing.T) {
	p := ProvideTokenValue("CONFIG", "some-config-value")
	if p.Token != "CONFIG" {
		t.Errorf("expected 'CONFIG', got %q", p.Token)
	}
	if p.Value != "some-config-value" {
		t.Error("unexpected value")
	}
}

func TestProvideFactory_PanicOnNonFunc(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	ProvideFactory[testServiceA]("not a function")
}
