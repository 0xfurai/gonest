package gonest

import "testing"

func TestReflector_SetAndGet(t *testing.T) {
	r := NewReflector()
	target := "handler1"

	r.Set(target, "roles", []string{"admin"})

	val, ok := r.Get(target, "roles")
	if !ok {
		t.Fatal("expected to find metadata")
	}
	roles := val.([]string)
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("expected [admin], got %v", roles)
	}
}

func TestReflector_GetMissing(t *testing.T) {
	r := NewReflector()
	_, ok := r.Get("nonexistent", "key")
	if ok {
		t.Error("expected not found")
	}
}

func TestReflector_GetAll(t *testing.T) {
	r := NewReflector()
	target := "handler2"

	r.Set(target, "roles", []string{"admin"})
	r.Set(target, "version", "v1")

	all := r.GetAll(target)
	if len(all) != 2 {
		t.Errorf("expected 2 metadata entries, got %d", len(all))
	}
	if all["version"] != "v1" {
		t.Errorf("expected v1, got %v", all["version"])
	}
}

func TestReflector_GetAll_NilTarget(t *testing.T) {
	r := NewReflector()
	all := r.GetAll("nonexistent")
	if all != nil {
		t.Error("expected nil for nonexistent target")
	}
}

func TestGetMetadata_Generic(t *testing.T) {
	r := NewReflector()
	target := "handler3"
	r.Set(target, "roles", []string{"admin", "user"})

	ctx := &executionContext{
		handler:  target,
		metadata: map[string]any{"roles": []string{"admin", "user"}},
	}

	roles, ok := GetMetadata[[]string](ctx, "roles")
	if !ok {
		t.Fatal("expected metadata found")
	}
	if len(roles) != 2 {
		t.Errorf("expected 2 roles, got %d", len(roles))
	}
}

func TestGetMetadata_WrongType(t *testing.T) {
	ctx := &executionContext{
		metadata: map[string]any{"count": 42},
	}

	_, ok := GetMetadata[string](ctx, "count")
	if ok {
		t.Error("expected type assertion to fail")
	}
}

func TestGetMetadata_Missing(t *testing.T) {
	ctx := &executionContext{
		metadata: map[string]any{},
	}

	_, ok := GetMetadata[string](ctx, "nonexistent")
	if ok {
		t.Error("expected not found")
	}
}
