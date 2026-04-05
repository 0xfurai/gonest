package graphql

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xfurai/gonest"
)

func TestEngine_QueryResolver(t *testing.T) {
	engine := NewEngine()
	engine.Query("hello", func(ctx *ResolverContext) (any, error) {
		return "Hello, World!", nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := makeCtx(w, r)

	resp := engine.Execute(ctx, Request{Query: "{ hello }"})
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}
	data := resp.Data.(map[string]any)
	if data["hello"] != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %v", data["hello"])
	}
}

func TestEngine_MutationResolver(t *testing.T) {
	engine := NewEngine()
	engine.Mutation("createCat", func(ctx *ResolverContext) (any, error) {
		return map[string]any{"id": 1, "name": "Pixel"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	ctx := makeCtx(w, r)

	resp := engine.Execute(ctx, Request{Query: "mutation { createCat }"})
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}
	data := resp.Data.(map[string]any)
	cat := data["createCat"].(map[string]any)
	if cat["name"] != "Pixel" {
		t.Errorf("expected 'Pixel', got %v", cat["name"])
	}
}

func TestEngine_UnknownField(t *testing.T) {
	engine := NewEngine()

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := makeCtx(w, r)

	resp := engine.Execute(ctx, Request{Query: "{ nonexistent }"})
	if len(resp.Errors) == 0 {
		t.Fatal("expected error for unknown field")
	}
}

func TestEngine_ResolverError(t *testing.T) {
	engine := NewEngine()
	engine.Query("fail", func(ctx *ResolverContext) (any, error) {
		return nil, gonest.NewInternalServerError("db error")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := makeCtx(w, r)

	resp := engine.Execute(ctx, Request{Query: "{ fail }"})
	if len(resp.Errors) == 0 {
		t.Fatal("expected error")
	}
	if resp.Errors[0].Message != "db error" {
		t.Errorf("expected 'db error', got %q", resp.Errors[0].Message)
	}
}

func TestEngine_ExplicitQueryKeyword(t *testing.T) {
	engine := NewEngine()
	engine.Query("cats", func(ctx *ResolverContext) (any, error) {
		return []string{"Pixel", "Luna"}, nil
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	ctx := makeCtx(w, r)

	resp := engine.Execute(ctx, Request{Query: "query { cats }"})
	if len(resp.Errors) > 0 {
		t.Fatalf("errors: %v", resp.Errors)
	}
}

func TestGraphQLModule_Integration(t *testing.T) {
	engine := NewEngine()
	engine.Query("greet", func(ctx *ResolverContext) (any, error) {
		return "Hello from GraphQL", nil
	})
	engine.Mutation("addItem", func(ctx *ResolverContext) (any, error) {
		return map[string]any{"id": 1}, nil
	})

	gqlModule := NewModule(Options{
		Path:       "/graphql",
		Playground: true,
	}, engine)

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{gqlModule},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Test query
	body := `{"query":"{ greet }"}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp.Data.(map[string]any)
	if data["greet"] != "Hello from GraphQL" {
		t.Errorf("expected greeting, got %v", data)
	}

	// Test playground
	req = httptest.NewRequest("GET", "/graphql", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("playground: expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "GraphQL Playground") {
		t.Error("expected playground HTML")
	}
}

func TestParseSimpleQuery(t *testing.T) {
	tests := []struct {
		query     string
		opType    string
		fieldName string
	}{
		{"{ hello }", "query", "hello"},
		{"query { cats }", "query", "cats"},
		{"mutation { createCat }", "mutation", "createCat"},
		{"query GetCats { cats }", "query", "cats"},
	}

	for _, tt := range tests {
		opType, fieldName := parseSimpleQuery(tt.query)
		if opType != tt.opType || fieldName != tt.fieldName {
			t.Errorf("parseSimpleQuery(%q): expected (%q, %q), got (%q, %q)",
				tt.query, tt.opType, tt.fieldName, opType, fieldName)
		}
	}
}

// helper
type simpleCtx struct {
	gonest.Context
}

func makeCtx(w http.ResponseWriter, r *http.Request) gonest.Context {
	return &simpleCtx{}
}
