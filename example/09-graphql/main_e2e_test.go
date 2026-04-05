package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/graphql"
)

// createTestApp builds a fresh app with the graphql module.
// The original main() creates these locally, so we recreate them here.
func createTestApp(t *testing.T) *gonest.Application {
	t.Helper()

	engine := graphql.NewEngine()

	engine.Query("recipes", func(ctx *graphql.ResolverContext) (any, error) {
		return recipes, nil
	})

	engine.Query("recipe", func(ctx *graphql.ResolverContext) (any, error) {
		return recipes[0], nil
	})

	engine.Mutation("addRecipe", func(ctx *graphql.ResolverContext) (any, error) {
		recipe := Recipe{
			ID:    len(recipes) + 1,
			Title: "New Recipe",
		}
		return recipe, nil
	})

	gqlModule := graphql.NewModule(graphql.Options{
		Path:       "/graphql",
		Playground: true,
	}, engine)

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{gqlModule},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	return app
}

func TestQueryRecipes(t *testing.T) {
	app := createTestApp(t)

	body := `{"query":"{ recipes }"}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatal("expected data in response")
	}
	if len(resp.Errors) > 0 {
		t.Errorf("unexpected errors: %v", resp.Errors)
	}
}

func TestMutationAddRecipe(t *testing.T) {
	app := createTestApp(t)

	body := `{"query":"mutation { addRecipe }"}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Fatal("expected data in response")
	}
}

func TestGraphQLPlayground(t *testing.T) {
	app := createTestApp(t)

	req := httptest.NewRequest("GET", "/graphql", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected HTML content type, got %q", ct)
	}

	if !strings.Contains(w.Body.String(), "GraphQL Playground") {
		t.Error("expected playground HTML in response")
	}
}

func TestGraphQLUnknownField(t *testing.T) {
	app := createTestApp(t)

	body := `{"query":"{ unknownField }"}`
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Errors) == 0 {
		t.Error("expected error for unknown field")
	}
}
