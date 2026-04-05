package graphql

import (
	"encoding/json"
	"net/http"

	"github.com/gonest"
)

// Options configures the GraphQL module.
type Options struct {
	// Path is the GraphQL endpoint (default: "/graphql").
	Path string
	// Playground enables the GraphQL playground at the endpoint (default: true).
	Playground bool
	// Schema is the GraphQL schema definition language string.
	Schema string
}

// Resolver defines how a field is resolved.
type Resolver struct {
	Name    string
	Handler ResolverFunc
}

// ResolverFunc resolves a GraphQL field.
type ResolverFunc func(ctx *ResolverContext) (any, error)

// ResolverContext provides access to query arguments and request context.
type ResolverContext struct {
	Args    map[string]any
	Context gonest.Context
	Info    FieldInfo
}

// FieldInfo describes the field being resolved.
type FieldInfo struct {
	FieldName  string
	ParentType string
}

// Request is a GraphQL request body.
type Request struct {
	Query         string         `json:"query"`
	OperationName string         `json:"operationName,omitempty"`
	Variables     map[string]any `json:"variables,omitempty"`
}

// Response is a GraphQL response body.
type Response struct {
	Data   any      `json:"data,omitempty"`
	Errors []GQLError `json:"errors,omitempty"`
}

// GQLError represents a GraphQL error.
type GQLError struct {
	Message string `json:"message"`
}

// Engine orchestrates GraphQL query execution.
// This is a simplified engine. For production, integrate with:
//   - github.com/99designs/gqlgen (code-first, recommended)
//   - github.com/graphql-go/graphql (schema-first)
type Engine struct {
	queries   map[string]ResolverFunc
	mutations map[string]ResolverFunc
}

// NewEngine creates a new GraphQL engine.
func NewEngine() *Engine {
	return &Engine{
		queries:   make(map[string]ResolverFunc),
		mutations: make(map[string]ResolverFunc),
	}
}

// Query registers a query resolver.
func (e *Engine) Query(name string, resolver ResolverFunc) {
	e.queries[name] = resolver
}

// Mutation registers a mutation resolver.
func (e *Engine) Mutation(name string, resolver ResolverFunc) {
	e.mutations[name] = resolver
}

// Execute runs a GraphQL query against registered resolvers.
// This is a simplified executor that matches top-level field names.
func (e *Engine) Execute(ctx gonest.Context, req Request) Response {
	// Parse the query to find the operation type and field
	opType, fieldName := parseSimpleQuery(req.Query)

	var resolver ResolverFunc
	switch opType {
	case "query":
		resolver = e.queries[fieldName]
	case "mutation":
		resolver = e.mutations[fieldName]
	}

	if resolver == nil {
		return Response{Errors: []GQLError{{Message: "field " + fieldName + " not found"}}}
	}

	rctx := &ResolverContext{
		Args:    req.Variables,
		Context: ctx,
		Info:    FieldInfo{FieldName: fieldName, ParentType: opType},
	}

	result, err := resolver(rctx)
	if err != nil {
		return Response{Errors: []GQLError{{Message: err.Error()}}}
	}

	return Response{Data: map[string]any{fieldName: result}}
}

// graphqlController handles GraphQL HTTP requests.
type graphqlController struct {
	engine     *Engine
	path       string
	playground bool
}

func (c *graphqlController) Register(r gonest.Router) {
	r.Post(c.path, c.handleQuery)
	if c.playground {
		r.Get(c.path, c.servePlayground)
	}
}

func (c *graphqlController) handleQuery(ctx gonest.Context) error {
	var req Request
	if err := ctx.Bind(&req); err != nil {
		return err
	}

	resp := c.engine.Execute(ctx, req)
	statusCode := http.StatusOK
	if len(resp.Errors) > 0 && resp.Data == nil {
		statusCode = http.StatusBadRequest
	}
	return ctx.JSON(statusCode, resp)
}

func (c *graphqlController) servePlayground(ctx gonest.Context) error {
	html := `<!DOCTYPE html>
<html>
<head>
  <title>GraphQL Playground</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
  <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
  <div id="root"></div>
  <script>
    window.addEventListener('load', function() {
      GraphQLPlayground.init(document.getElementById('root'), { endpoint: '` + c.path + `' })
    })
  </script>
</body>
</html>`
	ctx.ResponseWriter().Header().Set("Content-Type", "text/html; charset=utf-8")
	ctx.ResponseWriter().WriteHeader(http.StatusOK)
	_, err := ctx.ResponseWriter().Write([]byte(html))
	return err
}

// NewModule creates a GraphQL module.
func NewModule(opts Options, engine *Engine) *gonest.Module {
	if opts.Path == "" {
		opts.Path = "/graphql"
	}

	ctrl := &graphqlController{
		engine:     engine,
		path:       opts.Path,
		playground: opts.Playground,
	}

	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *graphqlController { return ctrl }},
		Providers:   []any{gonest.ProvideValue[*Engine](engine)},
		Exports:     []any{(*Engine)(nil)},
	})
}

// parseSimpleQuery extracts the operation type and first field name.
// Handles: "query { fieldName }" and "mutation { fieldName }"
// and "{ fieldName }" (implicit query)
func parseSimpleQuery(query string) (opType string, fieldName string) {
	query = trimSpaces(query)
	opType = "query"

	if len(query) > 8 && query[:5] == "query" {
		query = trimSpaces(query[5:])
	} else if len(query) > 11 && query[:8] == "mutation" {
		opType = "mutation"
		query = trimSpaces(query[8:])
	}

	// Skip operation name if present
	if query[0] != '{' {
		idx := indexOf(query, '{')
		if idx < 0 {
			return opType, ""
		}
		query = query[idx:]
	}

	// Extract first field name from { fieldName ... }
	if len(query) < 2 || query[0] != '{' {
		return opType, ""
	}
	query = trimSpaces(query[1:])

	end := 0
	for end < len(query) && query[end] != ' ' && query[end] != '(' && query[end] != '{' && query[end] != '}' {
		end++
	}

	return opType, query[:end]
}

func trimSpaces(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\r' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// ensure json import is used
var _ = json.Marshal
