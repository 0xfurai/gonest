package swagger

import (
	"reflect"
	"testing"
)

type TestCatDto struct {
	Name  string `json:"name" validate:"required" swagger:"example=Kitty"`
	Age   int    `json:"age" validate:"required,gte=0" swagger:"example=3"`
	Breed string `json:"breed" validate:"required" swagger:"example=Maine Coon"`
}

type TestCat struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Breed string `json:"breed"`
}

func TestGenerator_Generate(t *testing.T) {
	gen := NewGenerator(Options{
		Title:   "Cats API",
		Version: "1.0.0",
	})

	gen.AddRoute(RouteInfo{
		Method:      "GET",
		Path:        "/cats",
		Summary:     "Get all cats",
		Tags:        []string{"cats"},
		ResponseType: reflect.TypeOf([]TestCat{}),
	})

	gen.AddRoute(RouteInfo{
		Method:       "POST",
		Path:         "/cats",
		Summary:      "Create a cat",
		Tags:         []string{"cats"},
		RequestBody:  reflect.TypeOf(TestCatDto{}),
		ResponseType: reflect.TypeOf(TestCat{}),
		StatusCode:   201,
	})

	gen.AddRoute(RouteInfo{
		Method:       "GET",
		Path:         "/cats/:id",
		Summary:      "Get a cat by ID",
		Tags:         []string{"cats"},
		ResponseType: reflect.TypeOf(TestCat{}),
	})

	spec := gen.Generate()

	if spec.OpenAPI != "3.0.0" {
		t.Errorf("expected openapi 3.0.0, got %q", spec.OpenAPI)
	}
	if spec.Info.Title != "Cats API" {
		t.Errorf("expected 'Cats API', got %q", spec.Info.Title)
	}
	if len(spec.Paths) != 2 { // /cats and /cats/{id}
		t.Errorf("expected 2 paths, got %d", len(spec.Paths))
	}

	// Check /cats path
	catsPath, ok := spec.Paths["/cats"]
	if !ok {
		t.Fatal("expected /cats path")
	}
	if catsPath["get"] == nil {
		t.Error("expected GET /cats")
	}
	if catsPath["post"] == nil {
		t.Error("expected POST /cats")
	}

	// Check POST has request body
	postOp := catsPath["post"]
	if postOp.RequestBody == nil {
		t.Fatal("expected request body for POST /cats")
	}
	content, ok := postOp.RequestBody.Content["application/json"]
	if !ok {
		t.Fatal("expected application/json content")
	}
	if content.Schema == nil || content.Schema.Type != "object" {
		t.Error("expected object schema for request body")
	}

	// Check POST response is 201
	if _, ok := postOp.Responses["201"]; !ok {
		t.Error("expected 201 response for POST")
	}

	// Check /cats/{id} path
	catByIdPath, ok := spec.Paths["/cats/{id}"]
	if !ok {
		t.Fatal("expected /cats/{id} path")
	}
	getOp := catByIdPath["get"]
	if getOp == nil {
		t.Fatal("expected GET /cats/{id}")
	}
	// Should have path parameter
	hasIdParam := false
	for _, p := range getOp.Parameters {
		if p.Name == "id" && p.In == "path" {
			hasIdParam = true
		}
	}
	if !hasIdParam {
		t.Error("expected id path parameter")
	}
}

func TestConvertPathParams(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/cats", "/cats"},
		{"/cats/:id", "/cats/{id}"},
		{"/users/:userId/posts/:postId", "/users/{userId}/posts/{postId}"},
	}

	for _, tt := range tests {
		result := convertPathParams(tt.input)
		if result != tt.expected {
			t.Errorf("convertPathParams(%q): expected %q, got %q", tt.input, tt.expected, result)
		}
	}
}

func TestExtractPathParams(t *testing.T) {
	params := extractPathParams("/users/:userId/posts/:postId")
	if len(params) != 2 || params[0] != "userId" || params[1] != "postId" {
		t.Errorf("expected [userId postId], got %v", params)
	}
}

func TestTypeToSchema_Struct(t *testing.T) {
	gen := NewGenerator(Options{})
	schema := gen.typeToSchema(reflect.TypeOf(TestCatDto{}))

	if schema.Type != "object" {
		t.Errorf("expected object, got %q", schema.Type)
	}
	if len(schema.Properties) != 3 {
		t.Errorf("expected 3 properties, got %d", len(schema.Properties))
	}
	if schema.Properties["name"].Type != "string" {
		t.Error("expected name to be string")
	}
	if schema.Properties["age"].Type != "integer" {
		t.Error("expected age to be integer")
	}

	// Check required fields
	if len(schema.Required) != 3 {
		t.Errorf("expected 3 required fields, got %d: %v", len(schema.Required), schema.Required)
	}
}

func TestTypeToSchema_Slice(t *testing.T) {
	gen := NewGenerator(Options{})
	schema := gen.typeToSchema(reflect.TypeOf([]TestCat{}))

	if schema.Type != "array" {
		t.Errorf("expected array, got %q", schema.Type)
	}
	if schema.Items == nil || schema.Items.Type != "object" {
		t.Error("expected array of objects")
	}
}

func TestTypeToSchema_Primitives(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{"hello", "string"},
		{42, "integer"},
		{3.14, "number"},
		{true, "boolean"},
	}

	for _, tt := range tests {
		schema := goTypeToSchema(reflect.TypeOf(tt.input))
		if schema.Type != tt.expected {
			t.Errorf("for %T: expected %q, got %q", tt.input, tt.expected, schema.Type)
		}
	}
}

func TestSwaggerTag_Parsing(t *testing.T) {
	gen := NewGenerator(Options{})
	schema := gen.typeToSchema(reflect.TypeOf(TestCatDto{}))

	nameSchema := schema.Properties["name"]
	if nameSchema.Example != "Kitty" {
		t.Errorf("expected example 'Kitty', got %v", nameSchema.Example)
	}
}
