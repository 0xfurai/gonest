# GoNest

A progressive Go framework for building efficient, reliable, and scalable server-side applications. Inspired by NestJS, built idiomatically for Go.

## Installation

```bash
go get github.com/gonest
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    "github.com/gonest"
)

type AppController struct{}

func NewAppController() *AppController { return &AppController{} }

func (c *AppController) Register(r gonest.Router) {
    r.Get("/", c.hello)
}

func (c *AppController) hello(ctx gonest.Context) error {
    return ctx.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
    Controllers: []any{NewAppController},
})

func main() {
    app := gonest.Create(AppModule)
    log.Fatal(app.Listen(":3000"))
}
```

## Core Concepts

### Modules

Modules organize your application into cohesive blocks of functionality.

```go
var CatsModule = gonest.NewModule(gonest.ModuleOptions{
    Controllers: []any{NewCatsController},
    Providers:   []any{NewCatsService},
    Exports:     []any{(*CatsService)(nil)},
})

var AppModule = gonest.NewModule(gonest.ModuleOptions{
    Imports: []*gonest.Module{CatsModule},
})
```

### Controllers

Controllers handle incoming requests and return responses.

```go
type CatsController struct {
    service *CatsService
}

func NewCatsController(service *CatsService) *CatsController {
    return &CatsController{service: service}
}

func (c *CatsController) Register(r gonest.Router) {
    r.Prefix("/cats")
    r.Get("/", c.findAll)
    r.Get("/:id", c.findOne).Pipes(gonest.NewParseIntPipe("id"))
    r.Post("/", c.create)
}

func (c *CatsController) findAll(ctx gonest.Context) error {
    return ctx.JSON(http.StatusOK, c.service.FindAll())
}
```

### Providers (Services)

Providers encapsulate business logic and are injected via the DI container.

```go
type CatsService struct {
    cats []Cat
}

func NewCatsService() *CatsService {
    return &CatsService{}
}
```

### Dependency Injection

Dependencies are resolved automatically from constructor function signatures.

```go
// CatsController depends on CatsService
// The DI container inspects NewCatsController's parameters
// and resolves CatsService automatically.
func NewCatsController(service *CatsService) *CatsController {
    return &CatsController{service: service}
}
```

Provider types:
- `gonest.Provide(constructor)` — Constructor injection
- `gonest.ProvideValue[T](value)` — Pre-built value
- `gonest.Bind[Interface](constructor)` — Interface binding
- `gonest.ProvideToken(token, constructor)` — Token-based injection
- `gonest.ProvideWithScope(constructor, scope)` — Scoped provider

### Guards

Guards determine whether a request should proceed.

```go
type RolesGuard struct{}

func (g *RolesGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
    roles, ok := gonest.GetMetadata[[]string](ctx, "roles")
    if !ok {
        return true, nil
    }
    // Check user's role against required roles
    userRole := ctx.Header("X-Role")
    for _, r := range roles {
        if r == userRole {
            return true, nil
        }
    }
    return false, gonest.NewForbiddenException("insufficient permissions")
}

// Apply to route:
r.Post("/admin", c.adminOnly).
    SetMetadata("roles", []string{"admin"}).
    Guards(&RolesGuard{})
```

### Pipes

Pipes transform and validate input parameters.

Built-in pipes:
- `gonest.NewParseIntPipe(paramName)` — String to int
- `gonest.NewParseBoolPipe(paramName)` — String to bool
- `gonest.NewParseFloatPipe(paramName)` — String to float64
- `gonest.NewParseUUIDPipe(paramName)` — UUID validation
- `gonest.NewDefaultValuePipe(paramName, defaultValue)` — Default values
- `gonest.NewParseArrayPipe(paramName)` — Comma-separated to slice
- `gonest.NewValidationPipe()` — Struct validation

### Interceptors

Interceptors wrap handler execution for cross-cutting concerns.

```go
type LoggingInterceptor struct{}

func (i *LoggingInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
    start := time.Now()
    result, err := next.Handle()
    log.Printf("%s %s took %v", ctx.Method(), ctx.Path(), time.Since(start))
    return result, err
}
```

### Exception Filters

Exception filters handle errors thrown during request processing.

```go
type CustomFilter struct{}

func (f *CustomFilter) Catch(err error, host gonest.ArgumentsHost) error {
    httpCtx := host.SwitchToHTTP()
    resp := httpCtx.Response()
    // Custom error response
    return resp.Status(500).JSON(map[string]string{"error": err.Error()})
}
```

### Middleware

Middleware processes requests before routing.

```go
mw := gonest.MiddlewareFunc(func(ctx gonest.Context, next gonest.NextFunc) error {
    log.Printf("[%s] %s", ctx.Method(), ctx.Path())
    return next()
})
app.UseGlobalMiddleware(mw)
```

### Execution Pipeline

```
Request → Middleware → Guards → Interceptors (before) → Pipes → Handler → Interceptors (after) → Response
```

## Additional Modules

| Module | Import | Purpose |
|--------|--------|---------|
| Config | `gonest/config` | Environment variables and .env files |
| Cache | `gonest/cache` | In-memory caching with TTL |
| Schedule | `gonest/schedule` | Cron jobs, intervals, timeouts |
| Queue | `gonest/queue` | In-memory job queues |
| WebSocket | `gonest/websocket` | Real-time WebSocket gateway |
| Microservice | `gonest/microservice` | TCP, gRPC, NATS, Redis, Kafka transports |
| Swagger | `gonest/swagger` | OpenAPI documentation |
| GraphQL | `gonest/graphql` | GraphQL engine with playground |
| Health | `gonest/health` | Health check indicators and endpoint |
| Database | `gonest/database` | Generic Repository[T] + pagination |
| SQL | `gonest/database/sql` | PostgreSQL, MySQL, SQLite, SQL Server |
| MongoDB | `gonest/database/mongo` | MongoDB connection module |
| Testing | `gonest/testing` | Test module builder |

## Testing

```go
func TestCatsController(t *testing.T) {
    mod := testing.Test(CatsModule).
        OverrideProvider((*CatsService)(nil), NewMockCatsService).
        Compile(t)

    controller := testing.Resolve[*CatsController](mod)
    // Test controller methods...
}
```

## Examples

See the `example/` directory for complete applications:
- `01-cats-app` — REST API with guards, pipes, interceptors
- `02-websocket` — WebSocket gateway
- `03-microservice` — TCP microservice with HTTP gateway
- `04-auth-jwt` — JWT authentication
- `05-cache` — HTTP response caching
- `06-schedule` — Scheduled tasks
- `07-sse` — Server-Sent Events
- `08-config` — Configuration management

## License

MIT
