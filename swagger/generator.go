package swagger

import (
	"fmt"
	"reflect"
	"strings"
)

// OpenAPISpec represents an OpenAPI 3.0 specification.
type OpenAPISpec struct {
	OpenAPI    string                    `json:"openapi"`
	Info       Info                      `json:"info"`
	Paths      map[string]PathItem       `json:"paths"`
	Components *Components               `json:"components,omitempty"`
	Security   []map[string][]string     `json:"security,omitempty"`
}

type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type PathItem map[string]*Operation // method -> operation

type Operation struct {
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	OperationID string                `json:"operationId,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // path, query, header
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Required    bool                `json:"required,omitempty"`
	Description string              `json:"description,omitempty"`
	Content     map[string]MediaType `json:"content"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty"`
}

type Response struct {
	Description string              `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type Schema struct {
	Type       string             `json:"type,omitempty"`
	Format     string             `json:"format,omitempty"`
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Required   []string           `json:"required,omitempty"`
	Example    any                `json:"example,omitempty"`
	Ref        string             `json:"$ref,omitempty"`
}

type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// Generator builds OpenAPI specs from registered routes and struct types.
type Generator struct {
	opts       Options
	routes     []RouteInfo
	schemas    map[string]*Schema
}

// RouteInfo describes a route for documentation.
type RouteInfo struct {
	Method       string
	Path         string
	Summary      string
	Description  string
	Tags         []string
	RequestBody  reflect.Type
	ResponseType reflect.Type
	Parameters   []Parameter
	StatusCode   int
	Metadata     map[string]any
}

// NewGenerator creates a new OpenAPI generator.
func NewGenerator(opts Options) *Generator {
	return &Generator{
		opts:    opts,
		schemas: make(map[string]*Schema),
	}
}

// AddRoute adds a route to be documented.
func (g *Generator) AddRoute(info RouteInfo) {
	g.routes = append(g.routes, info)
}

// ConsumeRoute implements gonest.RouteConsumer so the generator automatically
// receives all registered routes after application initialization.
func (g *Generator) ConsumeRoute(method, path string, metadata map[string]any) {
	// Skip swagger's own routes
	if strings.HasPrefix(path, g.opts.Path) {
		return
	}

	info := RouteInfo{
		Method:   method,
		Path:     path,
		Metadata: metadata,
	}

	if summary, ok := metadata["summary"].(string); ok {
		info.Summary = summary
	}
	if tags, ok := metadata["tags"].([]string); ok {
		info.Tags = tags
	}
	if code, ok := metadata["__httpCode"].(int); ok {
		info.StatusCode = code
	}
	if body, ok := metadata["__body"]; ok && body != nil {
		info.RequestBody = reflect.TypeOf(body)
	}
	if resp, ok := metadata["__responseType"]; ok && resp != nil {
		info.ResponseType = reflect.TypeOf(resp)
	}

	if len(info.Tags) == 0 {
		if tag := deriveTag(path); tag != "" {
			info.Tags = []string{tag}
		}
	}

	g.routes = append(g.routes, info)
}

// deriveTag extracts a resource name from a URL path for auto-tagging.
// "/api/v1/users/:id" -> "users", "/cats" -> "cats"
func deriveTag(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segments) - 1; i >= 0; i-- {
		seg := segments[i]
		if seg == "" || strings.HasPrefix(seg, ":") || strings.HasPrefix(seg, "{") {
			continue
		}
		// Skip common prefixes
		if seg == "api" || seg == "v1" || seg == "v2" || seg == "v3" {
			continue
		}
		return seg
	}
	return ""
}

// Generate builds the OpenAPI spec from registered routes.
func (g *Generator) Generate() OpenAPISpec {
	spec := OpenAPISpec{
		OpenAPI: "3.0.0",
		Info: Info{
			Title:       g.opts.Title,
			Description: g.opts.Description,
			Version:     g.opts.Version,
		},
		Paths: make(map[string]PathItem),
	}

	for _, route := range g.routes {
		path := convertPathParams(route.Path)
		if _, ok := spec.Paths[path]; !ok {
			spec.Paths[path] = make(PathItem)
		}

		op := &Operation{
			Summary:     route.Summary,
			Description: route.Description,
			Tags:        route.Tags,
			Parameters:  route.Parameters,
			Responses:   make(map[string]Response),
			OperationID: strings.ToLower(route.Method) + strings.ReplaceAll(path, "/", "_"),
		}

		// Extract path parameters
		params := extractPathParams(route.Path)
		for _, p := range params {
			op.Parameters = append(op.Parameters, Parameter{
				Name:     p,
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			})
		}

		// Request body
		if route.RequestBody != nil {
			schema := g.typeToSchema(route.RequestBody)
			op.RequestBody = &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					"application/json": {Schema: schema},
				},
			}
		}

		// Success response
		statusCode := "200"
		if route.StatusCode > 0 {
			statusCode = fmt.Sprintf("%d", route.StatusCode)
		}
		resp := Response{Description: httpStatusText(statusCode)}
		if route.ResponseType != nil {
			schema := g.typeToSchema(route.ResponseType)
			resp.Content = map[string]MediaType{
				"application/json": {Schema: schema},
			}
		}
		op.Responses[statusCode] = resp

		// Common error responses based on method and path
		if route.Method == "POST" || route.Method == "PUT" || route.Method == "PATCH" {
			op.Responses["400"] = Response{Description: "Bad Request", Content: jsonErrorContent()}
		}
		if !isPublicRoute(route) {
			op.Responses["401"] = Response{Description: "Unauthorized", Content: jsonErrorContent()}
			// Add security requirement for protected routes
			if g.opts.BearerAuth {
				op.Security = []map[string][]string{{"bearer": {}}}
			}
		}
		if hasPathParam(route.Path) {
			op.Responses["404"] = Response{Description: "Not Found", Content: jsonErrorContent()}
		}

		method := strings.ToLower(route.Method)
		spec.Paths[path][method] = op
	}

	// Add the standard error schema to components
	spec.Components = &Components{
		Schemas: map[string]*Schema{
			"Error": {
				Type: "object",
				Properties: map[string]*Schema{
					"statusCode": {Type: "integer", Example: 400},
					"message":    {Type: "string", Example: "Bad Request"},
					"timestamp":  {Type: "string", Format: "date-time"},
					"path":       {Type: "string", Example: "/api/v1/resource"},
				},
			},
		},
	}
	for k, v := range g.schemas {
		spec.Components.Schemas[k] = v
	}

	if len(g.schemas) > 0 {
		spec.Components = &Components{Schemas: g.schemas}
	}

	// Add Bearer auth security scheme
	if g.opts.BearerAuth {
		if spec.Components == nil {
			spec.Components = &Components{}
		}
		if spec.Components.SecuritySchemes == nil {
			spec.Components.SecuritySchemes = make(map[string]*SecurityScheme)
		}
		spec.Components.SecuritySchemes["bearer"] = &SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "Enter your JWT token",
		}
	}

	return spec
}

func (g *Generator) typeToSchema(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Slice {
		return &Schema{
			Type:  "array",
			Items: g.typeToSchema(t.Elem()),
		}
	}
	if t.Kind() == reflect.Struct {
		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
		}
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if !field.IsExported() {
				continue
			}
			jsonTag := field.Tag.Get("json")
			name := field.Name
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "-" {
					name = parts[0]
				}
			}

			fieldSchema := goTypeToSchema(field.Type)

			// Parse swagger tag
			swaggerTag := field.Tag.Get("swagger")
			if swaggerTag != "" {
				parseSwaggerTag(swaggerTag, fieldSchema)
			}

			// Check validate tag for required
			validateTag := field.Tag.Get("validate")
			if strings.Contains(validateTag, "required") {
				schema.Required = append(schema.Required, name)
			}

			schema.Properties[name] = fieldSchema
		}
		return schema
	}
	return goTypeToSchema(t)
}

func goTypeToSchema(t reflect.Type) *Schema {
	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Slice:
		return &Schema{Type: "array", Items: goTypeToSchema(t.Elem())}
	default:
		return &Schema{Type: "object"}
	}
}

func parseSwaggerTag(tag string, schema *Schema) {
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "example":
			schema.Example = kv[1]
		case "description":
			// Store in a way that can be used
		case "format":
			schema.Format = kv[1]
		}
	}
}

func convertPathParams(path string) string {
	// Convert :param to {param}
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			segments[i] = "{" + seg[1:] + "}"
		}
	}
	return strings.Join(segments, "/")
}

func extractPathParams(path string) []string {
	var params []string
	segments := strings.Split(path, "/")
	for _, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			params = append(params, seg[1:])
		}
	}
	return params
}

func jsonErrorContent() map[string]MediaType {
	return map[string]MediaType{
		"application/json": {
			Schema: &Schema{Ref: "#/components/schemas/Error"},
		},
	}
}

func hasPathParam(path string) bool {
	return strings.Contains(path, ":")
}

func isPublicRoute(r RouteInfo) bool {
	if v, ok := r.Metadata["public"]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func httpStatusText(code string) string {
	switch code {
	case "200":
		return "OK"
	case "201":
		return "Created"
	case "204":
		return "No Content"
	default:
		return "Success"
	}
}

