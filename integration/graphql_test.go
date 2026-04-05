package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/graphql"
)

// ---------------------------------------------------------------------------
// GraphQL Integration Tests
// Mirror: original/integration/graphql-code-first/
// Mirror: original/integration/graphql-schema-first/
// ---------------------------------------------------------------------------

func createGraphQLApp(t *testing.T) *gonest.Application {
	t.Helper()

	engine := graphql.NewEngine()

	// Query resolvers
	engine.Query("recipes", func(ctx *graphql.ResolverContext) (any, error) {
		return []map[string]any{
			{"id": "1", "title": "Pizza", "description": "Italian classic"},
			{"id": "2", "title": "Sushi", "description": "Japanese delicacy"},
		}, nil
	})

	engine.Query("recipe", func(ctx *graphql.ResolverContext) (any, error) {
		id := ""
		if ctx.Args != nil {
			if v, ok := ctx.Args["id"]; ok {
				id, _ = v.(string)
			}
		}
		if id == "1" {
			return map[string]any{"id": "1", "title": "Pizza"}, nil
		}
		return nil, &graphqlTestError{msg: "recipe not found"}
	})

	// Mutation resolvers
	engine.Mutation("addRecipe", func(ctx *graphql.ResolverContext) (any, error) {
		return map[string]any{"id": "3", "title": "New Recipe"}, nil
	})

	engine.Mutation("removeRecipe", func(ctx *graphql.ResolverContext) (any, error) {
		return true, nil
	})

	gqlModule := graphql.NewModule(graphql.Options{
		Path:       "/graphql",
		Playground: true,
	}, engine)

	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{gqlModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	return app
}

type graphqlTestError struct {
	msg string
}

func (e *graphqlTestError) Error() string { return e.msg }

func gqlRequest(t *testing.T, app *gonest.Application, query string, variables map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body := graphql.Request{
		Query:     query,
		Variables: variables,
	}
	b, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Tests: Query
// ---------------------------------------------------------------------------

func TestGraphQL_QueryReturnsData(t *testing.T) {
	app := createGraphQLApp(t)
	defer app.Close()

	w := gqlRequest(t, app, `query { recipes }`, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data == nil {
		t.Fatal("expected data in response")
	}
	if len(resp.Errors) > 0 {
		t.Fatalf("unexpected errors: %v", resp.Errors)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be a map")
	}
	recipes, ok := data["recipes"].([]any)
	if !ok {
		t.Fatal("expected recipes to be an array")
	}
	if len(recipes) != 2 {
		t.Errorf("expected 2 recipes, got %d", len(recipes))
	}
}

func TestGraphQL_QueryNotFound(t *testing.T) {
	app := createGraphQLApp(t)
	defer app.Close()

	w := gqlRequest(t, app, `query { nonexistent }`, nil)

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Errors) == 0 {
		t.Error("expected error for nonexistent field")
	}
}

// ---------------------------------------------------------------------------
// Tests: Mutation
// ---------------------------------------------------------------------------

func TestGraphQL_MutationReturnsData(t *testing.T) {
	app := createGraphQLApp(t)
	defer app.Close()

	w := gqlRequest(t, app, `mutation { addRecipe }`, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data == nil {
		t.Fatal("expected data in response")
	}

	data := resp.Data.(map[string]any)
	recipe := data["addRecipe"].(map[string]any)
	if recipe["title"] != "New Recipe" {
		t.Errorf("expected New Recipe, got %v", recipe["title"])
	}
}

func TestGraphQL_MutationRemove(t *testing.T) {
	app := createGraphQLApp(t)
	defer app.Close()

	w := gqlRequest(t, app, `mutation { removeRecipe }`, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp.Data.(map[string]any)
	if data["removeRecipe"] != true {
		t.Errorf("expected true, got %v", data["removeRecipe"])
	}
}

// ---------------------------------------------------------------------------
// Tests: Playground
// ---------------------------------------------------------------------------

func TestGraphQL_PlaygroundServed(t *testing.T) {
	app := createGraphQLApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/graphql", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("expected text/html, got %q", w.Header().Get("Content-Type"))
	}
	if !strings.Contains(w.Body.String(), "GraphQL Playground") {
		t.Error("expected playground HTML")
	}
}

// ---------------------------------------------------------------------------
// Tests: Error handling in resolver
// ---------------------------------------------------------------------------

func TestGraphQL_ResolverError(t *testing.T) {
	app := createGraphQLApp(t)
	defer app.Close()

	w := gqlRequest(t, app, `query { recipe }`, map[string]any{"id": "999"})

	var resp graphql.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp.Errors) == 0 {
		t.Error("expected error from resolver")
	}
}

// ---------------------------------------------------------------------------
// Tests: Invalid JSON body
// ---------------------------------------------------------------------------

func TestGraphQL_InvalidBody(t *testing.T) {
	app := createGraphQLApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: GraphQL with guards
// Mirror: original/integration/graphql-code-first/e2e/guards-filters.spec.ts
// ---------------------------------------------------------------------------

func TestGraphQL_WithGlobalGuard(t *testing.T) {
	engine := graphql.NewEngine()
	engine.Query("protected", func(ctx *graphql.ResolverContext) (any, error) {
		return "secret data", nil
	})

	gqlModule := graphql.NewModule(graphql.Options{Path: "/graphql"}, engine)
	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{gqlModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalGuards(&authGuard{})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Without auth — blocked
	w := gqlRequest(t, app, `query { protected }`, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", w.Code)
	}

	// With auth — allowed
	body, _ := json.Marshal(graphql.Request{Query: `query { protected }`})
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/graphql", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token")
	app.Handler().ServeHTTP(w2, req)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 with auth, got %d", w2.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Engine injection via DI
// ---------------------------------------------------------------------------

func TestGraphQL_EngineResolvedFromDI(t *testing.T) {
	engine := graphql.NewEngine()
	engine.Query("ping", func(ctx *graphql.ResolverContext) (any, error) {
		return "pong", nil
	})

	gqlModule := graphql.NewModule(graphql.Options{Path: "/graphql"}, engine)
	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{gqlModule},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	resolved, err := gonest.Resolve[*graphql.Engine](app.GetContainer())
	if err != nil {
		t.Fatal(err)
	}
	if resolved != engine {
		t.Error("expected same engine instance from DI")
	}
}
