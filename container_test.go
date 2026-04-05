package gonest

import (
	"reflect"
	"testing"
)

type testServiceA struct {
	Value string
}

func newTestServiceA() *testServiceA {
	return &testServiceA{Value: "A"}
}

type testServiceB struct {
	A *testServiceA
}

func newTestServiceB(a *testServiceA) *testServiceB {
	return &testServiceB{A: a}
}

type testServiceC struct {
	A *testServiceA
	B *testServiceB
}

func newTestServiceC(a *testServiceA, b *testServiceB) *testServiceC {
	return &testServiceC{A: a, B: b}
}

func TestContainer_RegisterAndResolve(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))

	instance, err := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := instance.(*testServiceA)
	if svc.Value != "A" {
		t.Errorf("expected 'A', got %q", svc.Value)
	}
}

func TestContainer_Singleton(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))

	inst1, _ := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))
	inst2, _ := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))

	if inst1 != inst2 {
		t.Error("singleton scope should return same instance")
	}
}

func TestContainer_Transient(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(ProvideWithScope(newTestServiceA, ScopeTransient))

	inst1, _ := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))
	inst2, _ := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))

	if inst1 == inst2 {
		t.Error("transient scope should return different instances")
	}
}

func TestContainer_DependencyChain(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))
	c.Register(Provide(newTestServiceB))

	instance, err := c.Resolve(reflect.TypeOf((*testServiceB)(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := instance.(*testServiceB)
	if svc.A == nil || svc.A.Value != "A" {
		t.Error("expected B.A to be resolved")
	}
}

func TestContainer_DeepDependencyChain(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))
	c.Register(Provide(newTestServiceB))
	c.Register(Provide(newTestServiceC))

	instance, err := c.Resolve(reflect.TypeOf((*testServiceC)(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := instance.(*testServiceC)
	if svc.A == nil || svc.B == nil || svc.B.A == nil {
		t.Error("expected all dependencies to be resolved")
	}
	// A should be the same singleton
	if svc.A != svc.B.A {
		t.Error("expected shared singleton for A")
	}
}

func TestContainer_ValueProvider(t *testing.T) {
	c := NewContainer(NopLogger{})
	svc := &testServiceA{Value: "value-provided"}
	c.Register(ProvideValue[*testServiceA](svc))

	instance, err := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.(*testServiceA).Value != "value-provided" {
		t.Errorf("expected 'value-provided', got %q", instance.(*testServiceA).Value)
	}
}

func TestContainer_UnresolvedDependency(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceB)) // B depends on A, but A not registered

	_, err := c.Resolve(reflect.TypeOf((*testServiceB)(nil)))
	if err == nil {
		t.Fatal("expected error for unresolved dependency")
	}
}

type testRepository interface {
	Find() string
}

type testMemoryRepo struct{}

func (r *testMemoryRepo) Find() string { return "memory" }

func newTestMemoryRepo() *testMemoryRepo {
	return &testMemoryRepo{}
}

func TestContainer_InterfaceBinding(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Bind[testRepository](newTestMemoryRepo))

	instance, err := c.Resolve(reflect.TypeOf((*testRepository)(nil)).Elem())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo := instance.(testRepository)
	if repo.Find() != "memory" {
		t.Errorf("expected 'memory', got %q", repo.Find())
	}
}

func TestContainer_TokenProvider(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(ProvideTokenValue("DB_HOST", "localhost"))

	instance, err := c.ResolveByToken("DB_HOST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance != "localhost" {
		t.Errorf("expected 'localhost', got %v", instance)
	}
}

func TestContainer_TokenNotFound(t *testing.T) {
	c := NewContainer(NopLogger{})
	_, err := c.ResolveByToken("MISSING")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestContainer_ChildContainer(t *testing.T) {
	parent := NewContainer(NopLogger{})
	parent.Register(Provide(newTestServiceA))

	child := NewChildContainer(parent)
	child.Register(Provide(newTestServiceB))

	instance, err := child.Resolve(reflect.TypeOf((*testServiceB)(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	svc := instance.(*testServiceB)
	if svc.A == nil {
		t.Error("expected child to resolve A from parent")
	}
}

func TestContainer_Has(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))

	if !c.Has(reflect.TypeOf((*testServiceA)(nil))) {
		t.Error("expected Has to return true")
	}
	if c.Has(reflect.TypeOf((*testServiceB)(nil))) {
		t.Error("expected Has to return false")
	}
}

func TestContainer_ResolveGeneric(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))

	svc, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.Value != "A" {
		t.Errorf("expected 'A', got %q", svc.Value)
	}
}

func TestContainer_MustResolve(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))

	svc := MustResolve[*testServiceA](c)
	if svc.Value != "A" {
		t.Errorf("expected 'A', got %q", svc.Value)
	}
}

func TestContainer_MustResolve_Panics(t *testing.T) {
	c := NewContainer(NopLogger{})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		}
	}()
	MustResolve[*testServiceA](c)
}

func TestContainer_RegisterInstance(t *testing.T) {
	c := NewContainer(NopLogger{})
	svc := &testServiceA{Value: "direct"}
	c.RegisterInstance(reflect.TypeOf((*testServiceA)(nil)), svc)

	instance, err := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instance.(*testServiceA).Value != "direct" {
		t.Error("expected 'direct'")
	}
}

func TestContainer_ResolveAll(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(Provide(newTestServiceA))
	c.Register(ProvideValue[*testServiceB](&testServiceB{}))

	all, err := c.ResolveAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}

func TestContainer_RequestScope(t *testing.T) {
	c := NewContainer(NopLogger{})
	callCount := 0
	factory := func() *testServiceA {
		callCount++
		return &testServiceA{Value: Sprintf("request-%d", callCount)}
	}
	c.Register(ProvideWithScope(factory, ScopeRequest))

	// Each resolve should create a new instance for request scope
	inst1, _ := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))
	inst2, _ := c.Resolve(reflect.TypeOf((*testServiceA)(nil)))

	if inst1 == inst2 {
		t.Error("request scope should return different instances per resolve")
	}
	if callCount != 2 {
		t.Errorf("expected 2 constructor calls, got %d", callCount)
	}
}

func TestContainer_CreateRequestContainer(t *testing.T) {
	parent := NewContainer(NopLogger{})
	parent.Register(Provide(newTestServiceA)) // singleton in parent
	parent.Register(ProvideWithScope(func(a *testServiceA) *testServiceB {
		return &testServiceB{A: a}
	}, ScopeRequest))

	// Create two request containers
	req1 := parent.CreateRequestContainer()
	req2 := parent.CreateRequestContainer()

	b1, err := req1.Resolve(reflect.TypeOf((*testServiceB)(nil)))
	if err != nil {
		t.Fatalf("req1 resolve: %v", err)
	}
	b2, err := req2.Resolve(reflect.TypeOf((*testServiceB)(nil)))
	if err != nil {
		t.Fatalf("req2 resolve: %v", err)
	}

	// Different request containers should get different B instances
	if b1 == b2 {
		t.Error("different request containers should get different instances")
	}

	// But they should share the same singleton A from parent
	if b1.(*testServiceB).A != b2.(*testServiceB).A {
		t.Error("singleton A should be shared across request containers")
	}
}

func TestContainer_FactoryProvider(t *testing.T) {
	c := NewContainer(NopLogger{})
	c.Register(ProvideValue[string]("hello"))
	c.Register(ProvideFactory[*testServiceA](func() *testServiceA {
		return &testServiceA{Value: "from-factory"}
	}))

	svc, err := Resolve[*testServiceA](c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.Value != "from-factory" {
		t.Errorf("expected 'from-factory', got %q", svc.Value)
	}
}
