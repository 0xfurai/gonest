# GoNest

[![Website](https://img.shields.io/badge/website-gonest.org-blue)](https://gonest.org) [![Docs](https://img.shields.io/badge/docs-gonest.org%2Fdocs-green)](https://gonest.org/docs/introduction/)

**Website:** [https://gonest.org](https://gonest.org) | **Docs:** [https://gonest.org/docs/introduction/](https://gonest.org/docs/introduction/)

A progressive Go framework for building efficient, reliable and scalable server-side applications. Inspired by [NestJS](https://nestjs.com/), GoNest brings the same modular, dependency-injected architecture to the Go ecosystem.

```
go get github.com/0xfurai/gonest
```

**Go 1.23+ required.**

---

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    "github.com/0xfurai/gonest"
)

type GreetController struct{}

func NewGreetController() *GreetController { return &GreetController{} }

func (c *GreetController) Register(r gonest.Router) {
    r.Get("/hello", func(ctx gonest.Context) error {
        return ctx.JSON(http.StatusOK, map[string]string{"message": "Hello, GoNest!"})
    })
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
    Controllers: []any{NewGreetController},
})

func main() {
    app := gonest.Create(AppModule)
    log.Fatal(app.Listen(":3000"))
}
```

---

## Core Concepts

### Modules

```go
var CatsModule = gonest.NewModule(gonest.ModuleOptions{
    Imports:     []*gonest.Module{DatabaseModule},
    Controllers: []any{NewCatsController},
    Providers:   []any{NewCatsService},
    Exports:     []any{(*CatsService)(nil)},
})
```

### Controllers & Routing

```go
func (c *CatsController) Register(r gonest.Router) {
    r.Prefix("/cats")

    r.Get("/", c.findAll)
    r.Post("/", c.create).HttpCode(http.StatusCreated)
    r.Get("/:id", c.findOne).Pipes(gonest.NewParseIntPipe("id"))
}
```

### Providers & Dependency Injection

```go
gonest.Provide(NewCatsService)
gonest.ProvideValue[Logger](myLogger)
gonest.ProvideFactory[*DBConnection](func(config *ConfigService) *DBConnection {
    return Connect(config.Get("DATABASE_URL"))
})
gonest.Bind[Repository, SqlRepository]()
```

---

## Request Pipeline

```
Middleware -> Guards -> Interceptors (pre) -> Pipes -> Handler -> Interceptors (post) -> Exception Filters
```

### Middleware

```go
func (m *LoggerMiddleware) Use(ctx gonest.Context, next gonest.NextFunc) error {
    log.Printf("[%s] %s", ctx.Method(), ctx.Path())
    return next()
}

app.UseGlobalMiddleware(&LoggerMiddleware{})
```

### Guards

```go
func (g *RolesGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
    roles, ok := gonest.GetMetadata[[]string](ctx, "roles")
    if !ok {
        return true, nil
    }
    userRole := ctx.Header("X-User-Role")
    for _, r := range roles {
        if r == userRole { return true, nil }
    }
    return false, gonest.NewForbiddenException("insufficient permissions")
}
```

### Interceptors

```go
func (i *TimingInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
    start := time.Now()
    result, err := next.Handle()
    log.Printf("Request took %v", time.Since(start))
    return result, err
}
```

### Pipes

Built-in: `ParseIntPipe`, `ParseBoolPipe`, `ParseFloatPipe`, `ParseUUIDPipe`, `ParseDatePipe`, `ParseEnumPipe`, `ParseArrayPipe`, `DefaultValuePipe`, `ParseFilePipe`, `ValidationPipe`.

```go
r.Get("/:id", c.findOne).Pipes(gonest.NewParseIntPipe("id"))
app.UseGlobalPipes(gonest.NewValidationPipe())
```

### Exception Filters

```go
func (f *CustomFilter) Catch(err error, host gonest.ArgumentsHost) error {
    httpHost := host.SwitchToHTTP()
    httpHost.GetResponse().WriteHeader(500)
    json.NewEncoder(httpHost.GetResponse()).Encode(map[string]string{"error": err.Error()})
    return nil
}
```

---

## Features

**Built-in** — validation, configuration (`config/`), caching (`cache/`), scheduling (`schedule/`), queues (`queue/`), WebSockets (`websocket/`), SSE, event emitter, rate limiting, API versioning, serialization, file upload, sessions, CORS, templating, static file serving, host/subdomain routing.

**Database** — SQL (`database/sql/` — Postgres, MySQL, SQLite, SQL Server), MongoDB (`database/mongo/`), generic `Repository[T]` with pagination.

**Microservices** (`microservice/`) — TCP, gRPC, NATS, Redis, Kafka, RabbitMQ, MQTT, custom transports.

**GraphQL** (`graphql/`) — query/mutation engine with playground.

**Swagger / OpenAPI** (`swagger/`) — auto-generated spec + Swagger UI.

**Health Checks** (`health/`) — pluggable indicators + `/health` endpoint.

**Testing** (`testing/`) — test module builder with provider overrides.

**Advanced** — dynamic modules, configurable module builder, lazy module loading, discovery service, lifecycle hooks, reflection/metadata, application context, REPL, graceful shutdown.

**HTTP Exceptions** — built-in exceptions for common HTTP status codes (400–505) via `gonest.New<Name>Exception(msg)`.

**Platform Adapter** — ships with a `net/http` trie-based router (`platform/stdhttp/`). Implement `platform.HTTPAdapter` for a different router.

---

## Examples

See the `example/` directory for 26 example applications covering REST APIs, WebSockets, microservices, JWT auth, caching, scheduling, GraphQL, Swagger, databases, and more.

```bash
cd example/01-cats-app
go run main.go
```

---

## License

See [LICENSE](LICENSE) for details.
