package gonest

import (
	"net/http/httptest"
	"testing"
)

type UserEntity struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password" serialize:"exclude"`
	SSN      string `json:"ssn" serialize:"group=admin"`
	Role     string `json:"role" serialize:"expose"`
}

func TestSerializerInterceptor_ExcludeField(t *testing.T) {
	interceptor := NewSerializerInterceptor()

	user := UserEntity{
		ID: 1, Name: "John", Email: "john@example.com",
		Password: "secret123", SSN: "123-45-6789", Role: "admin",
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{})

	result, err := interceptor.Intercept(execCtx, NewCallHandler(func() (any, error) {
		return user, nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	m := result.(map[string]any)
	if _, ok := m["password"]; ok {
		t.Error("password should be excluded")
	}
	if _, ok := m["name"]; !ok {
		t.Error("name should be present")
	}
	if _, ok := m["role"]; !ok {
		t.Error("role (expose) should be present")
	}
}

func TestSerializerInterceptor_GroupHidden(t *testing.T) {
	interceptor := NewSerializerInterceptor()

	user := UserEntity{ID: 1, SSN: "123-45-6789"}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	// No groups set
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{})

	result, _ := interceptor.Intercept(execCtx, NewCallHandler(func() (any, error) {
		return user, nil
	}))

	m := result.(map[string]any)
	if _, ok := m["ssn"]; ok {
		t.Error("ssn should be hidden without admin group")
	}
}

func TestSerializerInterceptor_GroupVisible(t *testing.T) {
	interceptor := NewSerializerInterceptor()

	user := UserEntity{ID: 1, SSN: "123-45-6789"}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{
		"serialize_groups": []string{"admin"},
	})

	result, _ := interceptor.Intercept(execCtx, NewCallHandler(func() (any, error) {
		return user, nil
	}))

	m := result.(map[string]any)
	if _, ok := m["ssn"]; !ok {
		t.Error("ssn should be visible with admin group")
	}
}

func TestSerializerInterceptor_Slice(t *testing.T) {
	interceptor := NewSerializerInterceptor()

	users := []UserEntity{
		{ID: 1, Name: "A", Password: "pa"},
		{ID: 2, Name: "B", Password: "pb"},
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{})

	result, _ := interceptor.Intercept(execCtx, NewCallHandler(func() (any, error) {
		return users, nil
	}))

	arr := result.([]any)
	if len(arr) != 2 {
		t.Fatalf("expected 2, got %d", len(arr))
	}
	for i, item := range arr {
		m := item.(map[string]any)
		if _, ok := m["password"]; ok {
			t.Errorf("user %d: password should be excluded", i)
		}
	}
}

func TestSerializerInterceptor_NilResult(t *testing.T) {
	interceptor := NewSerializerInterceptor()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{})

	result, err := interceptor.Intercept(execCtx, NewCallHandler(func() (any, error) {
		return nil, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSerializerInterceptor_HandlerError(t *testing.T) {
	interceptor := NewSerializerInterceptor()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{})

	_, err := interceptor.Intercept(execCtx, NewCallHandler(func() (any, error) {
		return nil, NewInternalServerError("db failed")
	}))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSerializerInterceptor_PrimitivePassthrough(t *testing.T) {
	interceptor := NewSerializerInterceptor()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := newContext(w, r)
	execCtx := newExecutionContext(ctx, nil, nil, map[string]any{})

	result, _ := interceptor.Intercept(execCtx, NewCallHandler(func() (any, error) {
		return "just a string", nil
	}))
	if result != "just a string" {
		t.Errorf("expected passthrough, got %v", result)
	}
}
