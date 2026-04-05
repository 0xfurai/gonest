# GoNest

A progressive Go framework for building efficient, reliable and scalable server-side applications. Inspired by [NestJS](https://nestjs.com/), GoNest brings the same modular, dependency-injected architecture to the Go ecosystem.

```
go get github.com/gonest
```

**Go 1.23+ required.**

---

## Quick Start

```go
package main

import (
    "log"
    "net/http"

    "github.com/gonest"
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

## Table of Contents

- [Philosophy](#philosophy)
- [Core Concepts](#core-concepts)
  - [Modules](#modules)
  - [Controllers & Routing](#controllers--routing)
  - [Providers & Dependency Injection](#providers--dependency-injection)
  - [Scopes](#scopes)
- [Request Pipeline](#request-pipeline)
  - [Middleware](#middleware)
  - [Guards](#guards)
  - [Interceptors](#interceptors)
  - [Pipes](#pipes)
  - [Exception Filters](#exception-filters)
- [Built-in Features](#built-in-features)
  - [Validation](#validation)
  - [Configuration](#configuration)
  - [Caching](#caching)
  - [Scheduling](#scheduling)
  - [Queues](#queues)
  - [WebSockets](#websockets)
  - [Server-Sent Events (SSE)](#server-sent-events-sse)
  - [Event Emitter](#event-emitter)
  - [Rate Limiting / Throttle](#rate-limiting--throttle)
  - [API Versioning](#api-versioning)
  - [Serialization](#serialization)
  - [File Upload](#file-upload)
  - [Streaming Files](#streaming-files)
  - [Raw Body Access](#raw-body-access)
  - [Sessions](#sessions)
  - [CORS](#cors)
  - [Templating / MVC](#templating--mvc)
  - [Serve Static Files](#serve-static-files)
  - [Host / Subdomain Routing](#host--subdomain-routing)
- [Database](#database)
  - [SQL (Postgres, MySQL, SQLite, SQL Server)](#sql)
  - [MongoDB](#mongodb)
  - [Repository Pattern](#repository-pattern)
- [Microservices](#microservices)
- [GraphQL](#graphql)
- [Swagger / OpenAPI](#swagger--openapi)
- [Health Checks](#health-checks)
- [Testing](#testing)
- [Advanced](#advanced)
  - [Dynamic Modules](#dynamic-modules)
  - [Configurable Module Builder](#configurable-module-builder)
  - [Lazy Module Loading](#lazy-module-loading)
  - [Discovery Service](#discovery-service)
  - [Graph Inspector](#graph-inspector)
  - [Lifecycle Hooks](#lifecycle-hooks)
  - [Reflection & Metadata](#reflection--metadata)
  - [Application Context](#application-context)
  - [REPL](#repl)
  - [Graceful Shutdown](#graceful-shutdown)
- [HTTP Exceptions](#http-exceptions)
- [Platform Adapter](#platform-adapter)
- [Examples](#examples)
- [Project Structure](#project-structure)

---

## Philosophy

GoNest brings the proven architectural patterns of NestJS to Go:

- **Modular architecture** -- organize code into cohesive, self-contained modules
- **Dependency injection** -- constructor-based DI with automatic resolution and scope management
- **Request pipeline** -- composable middleware, guards, interceptors, pipes, and exception filters
- **Type safety** -- leverages Go generics throughout
- **Batteries included** -- configuration, caching, scheduling, queues, WebSockets, GraphQL, microservices, Swagger, and more
- **Extensible** -- custom transports, validators, serializers, and platform adapters

---

## Core Concepts

### Modules

Modules are the fundamental organizational unit. Every application has at least one root module.

```go
var CatsModule = gonest.NewModule(gonest.ModuleOptions{
    Imports:     []*gonest.Module{DatabaseModule},
    Controllers: []any{NewCatsController},
    Providers:   []any{NewCatsService},
    Exports:     []any{(*CatsService)(nil)},
})
```

| Field         | Description                                               |
|---------------|-----------------------------------------------------------|
| `Imports`     | Modules whose exported providers are needed in this module |
| `Controllers` | Controllers instantiated by this module                    |
| `Providers`   | Providers available for injection within this module       |
| `Exports`     | Providers made available to importing modules              |
| `Global`      | When `true`, makes all providers globally available        |

### Controllers & Routing

Controllers handle incoming requests and return responses. Register routes via the fluent `Router` interface.

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
    r.Post("/", c.create).HttpCode(http.StatusCreated)
    r.Get("/:id", c.findOne).Pipes(gonest.NewParseIntPipe("id"))
    r.Put("/:id", c.update).Pipes(gonest.NewParseIntPipe("id"))
    r.Delete("/:id", c.remove).Pipes(gonest.NewParseIntPipe("id"))
}
```

**Supported HTTP methods:** `Get`, `Post`, `Put`, `Delete`, `Patch`, `Options`, `Head`, `All`, plus WebDAV methods (`Search`, `Propfind`, `Proppatch`, `Mkcol`, `Copy`, `Move`, `Lock`, `Unlock`).

**Route builder** -- each route method returns a `RouteBuilder` with a fluent API:

```go
r.Post("/", c.create).
    SetMetadata("roles", []string{"admin"}).
    Guards(NewRolesGuard()).
    Interceptors(&LoggingInterceptor{}).
    Pipes(gonest.NewValidationPipe()).
    Filters(&CustomFilter{}).
    HttpCode(http.StatusCreated).
    Header("X-Custom", "value").
    Summary("Create a cat").
    Tags("cats").
    Body(CreateCatDto{}).
    Response(http.StatusCreated, Cat{})
```

### Providers & Dependency Injection

Providers are the core DI mechanism. The container resolves constructor dependencies automatically.

```go
// Constructor-based provider (most common)
gonest.Provide(NewCatsService)

// Value provider
gonest.ProvideValue[Logger](myLogger)

// Factory provider
gonest.ProvideFactory[*DBConnection](func(config *ConfigService) *DBConnection {
    return Connect(config.Get("DATABASE_URL"))
})

// Token-based provider
gonest.ProvideToken("API_KEY", NewAPIKeyProvider)
gonest.ProvideTokenValue("APP_NAME", "my-app")

// Interface binding
gonest.Bind[Repository, SqlRepository]()

// Optional dependencies
gonest.Optional(NewOptionalService)

// Forward references (circular dependencies)
gonest.ForwardRef(func() any { return NewServiceA })
```

**Child containers** support hierarchical resolution with parent fallback. **Request-scoped containers** are created per request for request-scoped providers.

### Scopes

Providers support three lifetime scopes:

| Scope       | Description                                    |
|-------------|------------------------------------------------|
| `Singleton` | Single shared instance (default)               |
| `Request`   | New instance per request                       |
| `Transient` | New instance every time it is resolved         |

```go
gonest.ProvideWithScope(NewRequestService, gonest.RequestScope)
```

Scope propagation: if a singleton depends on a request-scoped provider, the singleton is automatically elevated to request scope.

---

## Request Pipeline

Every request flows through the pipeline in this order:

```
Middleware -> Guards -> Interceptors (pre) -> Pipes -> Handler -> Interceptors (post) -> Exception Filters
```

### Middleware

Middleware runs before guards. Use it for logging, authentication, request transformation, etc.

```go
// Interface-based
type LoggerMiddleware struct{}

func (m *LoggerMiddleware) Use(ctx gonest.Context, next gonest.NextFunc) error {
    log.Printf("[%s] %s", ctx.Method(), ctx.Path())
    return next()
}

// Function-based
mw := gonest.MiddlewareFunc(func(ctx gonest.Context, next gonest.NextFunc) error {
    ctx.SetHeader("X-Request-ID", uuid.New().String())
    return next()
})
```

**Global middleware:**

```go
app.UseGlobalMiddleware(&LoggerMiddleware{})
```

**Module-scoped middleware** via `MiddlewareConfigurer`:

```go
func (s *AppService) Configure(consumer gonest.MiddlewareConsumer) {
    consumer.Apply(&AuthMiddleware{}).
        Exclude("/health", "/public/*").
        ForRoutes("/api/*")
}
```

### Guards

Guards determine whether a request should be handled. Return `(true, nil)` to allow, `(false, error)` to reject.

```go
type RolesGuard struct{}

func (g *RolesGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
    roles, ok := gonest.GetMetadata[[]string](ctx, "roles")
    if !ok {
        return true, nil // no roles required
    }
    userRole := ctx.Header("X-User-Role")
    for _, r := range roles {
        if r == userRole {
            return true, nil
        }
    }
    return false, gonest.NewForbiddenException("insufficient permissions")
}
```

Apply per-route or globally:

```go
r.Post("/", c.create).Guards(&RolesGuard{}).SetMetadata("roles", []string{"admin"})

app.UseGlobalGuards(&AuthGuard{})
```

### Interceptors

Interceptors wrap the execution of a handler, allowing pre- and post-processing.

```go
type TimingInterceptor struct{}

func (i *TimingInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
    start := time.Now()
    result, err := next.Handle()
    log.Printf("Request took %v", time.Since(start))
    return result, err
}
```

Apply per-route or globally:

```go
r.Get("/", c.findAll).Interceptors(&TimingInterceptor{})

app.UseGlobalInterceptors(&TimingInterceptor{})
```

### Pipes

Pipes transform and validate input values (path params, query params, body).

**Built-in pipes:**

| Pipe                 | Description                                     |
|----------------------|-------------------------------------------------|
| `ParseIntPipe`       | Convert string param to `int`                   |
| `ParseBoolPipe`      | Convert string param to `bool`                  |
| `ParseFloatPipe`     | Convert string param to `float64`               |
| `ParseUUIDPipe`      | Validate UUID format                            |
| `ParseDatePipe`      | Parse date (RFC3339 or YYYY-MM-DD)              |
| `ParseEnumPipe`      | Validate value against allowed enum values       |
| `ParseArrayPipe`     | Split comma-separated string into array          |
| `DefaultValuePipe`   | Provide a default if value is empty              |
| `ParseFilePipe`      | Validate uploaded files                          |
| `ValidationPipe`     | Validate struct fields via tags                  |

```go
r.Get("/:id", c.findOne).Pipes(gonest.NewParseIntPipe("id"))
r.Get("/:date", c.byDate).Pipes(gonest.NewParseDatePipe("date"))

app.UseGlobalPipes(gonest.NewValidationPipe())
```

### Exception Filters

Exception filters handle errors thrown during request processing.

```go
type CustomFilter struct{}

func (f *CustomFilter) Catch(err error, host gonest.ArgumentsHost) error {
    httpHost := host.SwitchToHTTP()
    httpHost.GetResponse().WriteHeader(500)
    json.NewEncoder(httpHost.GetResponse()).Encode(map[string]string{
        "error": err.Error(),
    })
    return nil
}
```

The `DefaultExceptionFilter` is built-in and returns JSON error responses automatically. Override per-route or globally:

```go
r.Post("/", c.create).Filters(&CustomFilter{})

app.UseGlobalFilters(&CustomFilter{})
```

---

## Built-in Features

### Validation

The `ValidationPipe` validates struct fields using `validate` tags:

```go
type CreateCatDto struct {
    Name  string `json:"name"  validate:"required"`
    Age   int    `json:"age"   validate:"required,gte=0,lte=30"`
    Email string `json:"email" validate:"email"`
    Breed string `json:"breed" validate:"required,min=2,max=50"`
}
```

**Supported rules:** `required`, `min=N`, `max=N`, `gte=N`, `lte=N`, `email`, `omitempty`.

Apply globally or per-route:

```go
app.UseGlobalPipes(gonest.NewValidationPipe())
```

### Configuration

`config/` -- Environment variable and `.env` file support.

```go
import "github.com/gonest/config"

// Register in module
var AppModule = gonest.NewModule(gonest.ModuleOptions{
    Imports: []*gonest.Module{config.NewModule()},
})

// Inject and use
type AppService struct {
    config *config.ConfigService
}

func (s *AppService) GetPort() string {
    return s.config.GetOrDefault("PORT", "3000")
}
```

| Method                | Description                      |
|-----------------------|----------------------------------|
| `Get(key)`            | Get string value                 |
| `GetOrDefault(k, d)`  | Get with fallback                |
| `GetInt(key)`         | Get int value                    |
| `GetIntOrDefault(k,d)`| Get int with fallback            |
| `GetBool(key)`        | Get bool value                   |
| `GetBoolOrDefault(k,d)`| Get bool with fallback          |
| `Has(key)`            | Check key existence              |
| `Set(key, value)`     | Set value at runtime             |

### Caching

`cache/` -- In-memory cache with TTL and automatic cleanup.

```go
import "github.com/gonest/cache"

store := cache.NewMemoryStore(5 * time.Minute) // TTL

store.Set("key", value, 10*time.Minute)
val, found := store.Get("key")
store.Delete("key")
store.Clear()
```

**CacheInterceptor** -- automatically cache GET responses:

```go
r.Get("/cats", c.findAll).
    Interceptors(cache.NewCacheInterceptor(store, 5*time.Minute))
```

### Scheduling

`schedule/` -- Cron, interval, and timeout job scheduling.

```go
import "github.com/gonest/schedule"

scheduler := schedule.NewScheduler()

// Run every 5 seconds
scheduler.AddInterval("cleanup", 5*time.Second, func() {
    log.Println("Running cleanup...")
})

// Run once after 10 seconds
scheduler.AddTimeout("init", 10*time.Second, func() {
    log.Println("Initialization complete")
})

// Cron expression
scheduler.AddCron("report", "0 9 * * *", func() {
    log.Println("Daily report")
})

scheduler.Start()
defer scheduler.Stop()
```

### Queues

`queue/` -- In-memory job queue with configurable workers and retries.

```go
import "github.com/gonest/queue"

q := queue.NewQueue("emails", 100) // buffer size

q.Process(3, func(job *queue.Job) error { // 3 workers
    return sendEmail(job.Data)
})

q.Add("send-welcome", userData, queue.JobOptions{MaxRetries: 3})
```

### WebSockets

`websocket/` -- WebSocket gateway pattern.

```go
import "github.com/gonest/websocket"

type ChatGateway struct{}

func (g *ChatGateway) Handlers() map[string]websocket.MessageHandler {
    return map[string]websocket.MessageHandler{
        "message": g.handleMessage,
    }
}

func (g *ChatGateway) OnConnection(client *websocket.Client) {
    log.Println("Client connected:", client)
}

func (g *ChatGateway) OnDisconnect(client *websocket.Client) {
    log.Println("Client disconnected:", client)
}

func (g *ChatGateway) handleMessage(client *websocket.Client, msg websocket.Message) {
    client.Send(websocket.OutgoingMessage{Event: "reply", Data: msg.Data})
}
```

### Server-Sent Events (SSE)

```go
r.Get("/events", gonest.SSE(func(stream *gonest.SSEStream, ctx gonest.Context) {
    for i := 0; i < 10; i++ {
        stream.Send(gonest.SSEEvent{
            Event: "tick",
            Data:  fmt.Sprintf("count: %d", i),
        })
        time.Sleep(time.Second)
    }
    stream.Close()
}))
```

### Event Emitter

Pub/sub event system for decoupled communication between providers.

```go
emitter := gonest.NewEventEmitter()

emitter.On("user.created", func(data any) {
    user := data.(User)
    log.Println("Welcome", user.Name)
})

emitter.Emit("user.created", newUser)
emitter.EmitAsync("user.created", newUser) // non-blocking
```

### Rate Limiting / Throttle

Token bucket rate limiter per IP.

```go
// 10 requests per 60 seconds per IP
throttle := gonest.NewThrottleGuard(10, 60*time.Second)

app.UseGlobalGuards(throttle)

// Or per-route with metadata-based configuration
r.Get("/sensitive", c.handler).Guards(&gonest.ThrottleByMetadataGuard{})
```

### API Versioning

Support for URI, Header, MediaType, and Custom versioning strategies.

```go
app.UseGlobalMiddleware(gonest.VersioningMiddleware(gonest.VersioningOptions{
    Type:          gonest.URIVersioning,
    DefaultVersion: "1",
}))

// Restrict route to version
r.Get("/cats", c.findAll).Guards(gonest.VersionGuard("2"))

// Version-neutral route (matches all versions)
r.Get("/health", c.health).Guards(gonest.VersionGuard(gonest.VersionNeutral))
```

| Type              | Extracts version from              |
|-------------------|------------------------------------|
| `URIVersioning`   | URL path prefix (`/v1/cats`)       |
| `HeaderVersioning`| Custom header                       |
| `MediaTypeVersioning` | Accept header media type        |
| `CustomVersioning`| User-defined `VersionExtractor`    |

### Serialization

Transform response objects using struct tags.

```go
type UserResponse struct {
    ID       int    `json:"id"    serialize:"expose"`
    Name     string `json:"name"  serialize:"expose"`
    Email    string `json:"email" serialize:"expose,group=admin"`
    Password string `json:"password" serialize:"exclude"`
}

// Apply interceptor
r.Get("/users", c.findAll).
    Interceptors(&gonest.SerializerInterceptor{}).
    SetMetadata("serialize_groups", []string{"admin"})
```

| Tag                 | Description                             |
|---------------------|-----------------------------------------|
| `serialize:"expose"`| Always include this field               |
| `serialize:"exclude"`| Never include this field               |
| `serialize:"group=X"`| Include only when group X is active    |

### File Upload

```go
r.Post("/upload", c.upload).
    Interceptors(gonest.FileInterceptor("file"))

func (c *Controller) upload(ctx gonest.Context) error {
    file := gonest.GetUploadedFile(ctx)
    // file.Filename, file.Size, file.Header, file.File
    return ctx.JSON(http.StatusOK, map[string]string{"name": file.Filename})
}
```

**File validation with `ParseFilePipe`:**

```go
pipe := gonest.NewParsFilePipeBuilder().
    AddFileTypeValidator(".jpg", ".png", ".gif").
    AddFileSizeValidator(5 * 1024 * 1024). // 5MB
    Build()
```

### Streaming Files

Stream file downloads to the client.

```go
func (c *Controller) download(ctx gonest.Context) error {
    file := gonest.NewStreamableFile(reader).
        WithContentType("application/pdf").
        WithFileName("report.pdf").
        WithLength(fileSize)
    return file.Send(ctx)
}

// From bytes
file := gonest.NewStreamableFileFromBytes(data).
    WithContentType("image/png")
```

### Raw Body Access

```go
app.UseGlobalMiddleware(gonest.RawBodyMiddleware())

func (c *Controller) webhook(ctx gonest.Context) error {
    body := gonest.RawBody(ctx)
    // body is []byte of the original request body
}
```

### Sessions

Cookie-based session management with pluggable stores.

```go
store := gonest.NewMemorySessionStore()
app.SetSessionStore(store)
app.UseGlobalMiddleware(gonest.SessionMiddleware(gonest.SessionOptions{
    CookieName: "session_id",
    MaxAge:     24 * time.Hour,
    Secure:     true,
    HttpOnly:   true,
}))

func (c *Controller) handler(ctx gonest.Context) error {
    session := gonest.GetSession(ctx)
    session.SetValue("user_id", 42)
    userID := session.GetValue("user_id")
    session.Delete("user_id")
}
```

### CORS

```go
app.EnableCors(gonest.CorsOptions{
    Origin:      "https://example.com",
    Methods:     "GET, POST, PUT, DELETE",
    Headers:     "Content-Type, Authorization",
    Credentials: true,
})
```

### Templating / MVC

Render HTML templates using Go's `html/template`.

```go
engine := gonest.NewGoTemplateEngine("./views")
// or from embedded FS
engine := gonest.NewGoTemplateEngineFromFS(viewsFS, "views")

app.SetViewEngine(engine)

// In controller
r.Get("/home", c.home)

func (c *Controller) home(ctx gonest.Context) error {
    return gonest.Render(ctx, "home.html", map[string]any{
        "Title": "Welcome",
    })
}
```

### Serve Static Files

Serve static assets from a directory. See `example/15-serve-static`.

### Host / Subdomain Routing

Route based on the request `Host` header with parameter extraction.

```go
type TenantController struct{}

func (c *TenantController) Host() string {
    return ":tenant.example.com"
}

func (c *TenantController) Register(r gonest.Router) {
    r.Get("/", func(ctx gonest.Context) error {
        tenant := gonest.HostParam(ctx, "tenant")
        return ctx.JSON(200, map[string]string{"tenant": tenant})
    })
}
```

---

## Database

### SQL

`database/sql/` -- Generic SQL module supporting Postgres, MySQL, SQLite, and SQL Server.

```go
import dbsql "github.com/gonest/database/sql"

var DbModule = dbsql.NewModule(dbsql.Options{
    Driver:   dbsql.Postgres,
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "secret",
    Database: "mydb",
    MaxOpenConns:    25,
    MaxIdleConns:    5,
    ConnMaxLifetime: 5 * time.Minute,
})
```

Supported drivers: `Postgres`, `MySQL`, `SQLite`, `SQLServer`.

### MongoDB

`database/mongo/` -- MongoDB module with schema and index definitions.

```go
import "github.com/gonest/database/mongo"

var MongoModule = mongo.NewModule(mongo.Options{
    Host:     "localhost",
    Port:     27017,
    Database: "mydb",
})
```

### Repository Pattern

`database/` -- Generic `Repository[T]` interface for CRUD operations.

```go
import "github.com/gonest/database"

type CatRepository interface {
    database.Repository[Cat]
}

// Repository[T] provides:
// FindAll(ctx) ([]T, error)
// FindByID(ctx, id) (*T, error)
// Create(ctx, entity *T) error
// Update(ctx, entity *T) error
// Delete(ctx, id) error
// Count(ctx) (int64, error)
```

**Pagination helper:**

```go
result := database.Paginate[Cat](items, total, page, limit)
// result.Data, result.Total, result.Page, result.Limit, result.TotalPages
```

---

## Microservices

`microservice/` -- Transport abstraction for building distributed systems.

**Supported transports:**

| Transport   | Package                               |
|-------------|---------------------------------------|
| TCP         | `microservice/tcp`                    |
| gRPC        | `microservice/grpc`                   |
| NATS        | `microservice/nats`                   |
| Redis       | `microservice/redis`                  |
| Kafka       | `microservice/kafka`                  |
| RabbitMQ    | `microservice/rabbitmq`               |
| MQTT        | `microservice/mqtt`                   |
| Custom      | Implement `CustomTransportStrategy`   |

```go
import (
    "github.com/gonest"
    "github.com/gonest/microservice"
    "github.com/gonest/microservice/tcp"
)

// Server
app := gonest.CreateMicroservice(AppModule, microservice.ServerOptions{
    Transport: microservice.TCP,
    Options:   tcp.ServerConfig{Host: "0.0.0.0", Port: 4000},
})

// Message handler
type MathController struct{}

func (c *MathController) Patterns() map[string]microservice.MessageHandler {
    return map[string]microservice.MessageHandler{
        "sum": c.sum,
    }
}

// Client
client := tcp.NewClient(tcp.ClientConfig{Host: "localhost", Port: 4000})
result, err := client.Send("sum", map[string]any{"a": 1, "b": 2})
```

---

## GraphQL

`graphql/` -- GraphQL engine with query/mutation support and playground UI.

```go
import "github.com/gonest/graphql"

var GqlModule = graphql.Module(graphql.Options{
    Schema: `
        type Query {
            hello: String
        }
    `,
    Playground: true,
})

type HelloResolver struct{}

func (r *HelloResolver) Query() map[string]graphql.ResolverFunc {
    return map[string]graphql.ResolverFunc{
        "hello": func(ctx graphql.ResolverContext) (any, error) {
            return "Hello, GraphQL!", nil
        },
    }
}
```

---

## Swagger / OpenAPI

`swagger/` -- Automatic OpenAPI spec generation with Swagger UI.

```go
import "github.com/gonest/swagger"

var SwaggerModule = swagger.Module(swagger.Options{
    Title:       "Cats API",
    Description: "The cats API description",
    Version:     "1.0",
    Path:        "/swagger",
    BearerAuth:  true,
})

// Document routes using route builder
r.Post("/cats", c.create).
    Summary("Create a cat").
    Tags("cats").
    Body(CreateCatDto{}).
    Response(http.StatusCreated, Cat{})
```

Access the UI at `/swagger` and the JSON spec at `/swagger/json`.

---

## Health Checks

`health/` -- Health check indicators with automatic `/health` endpoint.

```go
import "github.com/gonest/health"

svc := health.NewHealthService()

// Built-in ping indicator
svc.AddIndicator(health.PingIndicator("api"))

// Custom indicator
svc.AddIndicator(health.CustomIndicator("database", func() health.HealthResult {
    if err := db.Ping(); err != nil {
        return health.HealthResult{Status: health.Down, Error: err.Error()}
    }
    return health.HealthResult{Status: health.Up}
}))
```

Response format:

```json
{
  "status": "up",
  "details": {
    "api": {"status": "up"},
    "database": {"status": "up"}
  }
}
```

---

## Testing

`testing/` -- Test module builder with provider overrides.

```go
import gotest "github.com/gonest/testing"

func TestCatsService(t *testing.T) {
    app := gotest.NewTestModuleBuilder().
        WithModule(CatsModule).
        OverrideProvider(NewCatsRepository, mockRepo).
        Build()

    // Use app.Handler() with httptest
    srv := httptest.NewServer(app.Handler())
    defer srv.Close()
}
```

You can also use `app.Init()` to compile modules without starting the server, then use `app.Handler()` directly with `httptest.NewRecorder`.

---

## Advanced

### Dynamic Modules

Modules that accept configuration at import time.

```go
var DatabaseModule = gonest.NewDynamicModule(gonest.ModuleOptions{
    Providers: []any{NewDatabaseConnection},
    Exports:   []any{(*DatabaseConnection)(nil)},
})

// ForRoot / ForFeature pattern
gonest.ForRoot[DatabaseOptions](DatabaseModule, opts)
gonest.ForFeature[FeatureOptions](FeatureModule, opts)
```

### Configurable Module Builder

Fluent API for building modules that accept configuration, including async factory patterns.

```go
builder := gonest.NewConfigurableModuleBuilder[MyConfig]()
builder.SetGlobal(true)

// Synchronous
mod := builder.Build(MyConfig{Port: 3000})

// Async with factory (dependencies injected)
mod := builder.BuildAsync(gonest.AsyncModuleOptions[MyConfig]{
    Imports: []*gonest.Module{ConfigModule},
    Factory: func(cfg *config.ConfigService) MyConfig {
        return MyConfig{Port: cfg.GetIntOrDefault("PORT", 3000)}
    },
})
```

### Lazy Module Loading

Load modules on demand at runtime.

```go
loader := app.GetLazyModuleLoader()
lazyMod, err := loader.Load(SomeModule)

service := gonest.LazyModuleResolve[*SomeService](lazyMod)
```

### Discovery Service

Runtime introspection of the module graph.

```go
discovery := app.GetDiscoveryService()

providers := discovery.GetProviders()
controllers := discovery.GetControllers()
modules := discovery.GetModules()
```

### Graph Inspector

Analyze the dependency graph of the application.

```go
inspector := app.GetGraphInspector()
```

### Lifecycle Hooks

Implement these interfaces on providers to hook into the application lifecycle:

| Hook                         | When                                    |
|------------------------------|-----------------------------------------|
| `OnModuleInit`               | After module providers are resolved     |
| `OnModuleDestroy`            | Module teardown                         |
| `OnApplicationBootstrap`     | All modules initialized                 |
| `BeforeApplicationShutdown`  | Before shutdown begins                  |
| `OnApplicationShutdown`      | Application shutdown                    |

```go
type MyService struct{}

func (s *MyService) OnModuleInit() error {
    log.Println("Module initialized")
    return nil
}

func (s *MyService) OnApplicationShutdown(signal string) error {
    log.Printf("Shutting down due to %s", signal)
    return nil
}
```

### Reflection & Metadata

Store and retrieve metadata on handlers for use by guards and interceptors.

```go
reflector := gonest.NewReflector()
reflector.Set(handlerID, "roles", []string{"admin"})

roles, ok := reflector.Get(handlerID, "roles")

// Generic typed retrieval from ExecutionContext
roles, ok := gonest.GetMetadata[[]string](ctx, "roles")
```

### Application Context

Non-HTTP application context for CLI tools, workers, and background services.

```go
ctx := gonest.CreateApplicationContext(AppModule)
ctx.Init()

service, _ := ctx.Resolve(reflect.TypeOf((*MyService)(nil)))
service.(*MyService).DoWork()

ctx.Close()
```

### REPL

Interactive debugging shell for inspecting a running application.

```go
repl := gonest.NewREPL(app)
repl.Start() // interactive prompt

// Commands: modules, providers, controllers, routes, inspect, etc.
```

### Graceful Shutdown

```go
app.EnableShutdownHooks(syscall.SIGINT, syscall.SIGTERM)
app.ListenAndServeWithGracefulShutdown(":3000")
```

The application will:
1. Catch the OS signal
2. Run `BeforeApplicationShutdown` hooks on all providers
3. Stop accepting new connections and drain existing ones
4. Run `OnApplicationShutdown` hooks on all providers
5. Destroy all modules

---

## HTTP Exceptions

Built-in exception types for common HTTP error responses:

| Exception                            | Status |
|--------------------------------------|--------|
| `NewBadRequestException(msg)`        | 400    |
| `NewUnauthorizedException(msg)`      | 401    |
| `NewForbiddenException(msg)`         | 403    |
| `NewNotFoundException(msg)`          | 404    |
| `NewMethodNotAllowedException(msg)`  | 405    |
| `NewNotAcceptableException(msg)`     | 406    |
| `NewRequestTimeoutException(msg)`    | 408    |
| `NewConflictException(msg)`          | 409    |
| `NewGoneException(msg)`             | 410    |
| `NewPreconditionFailedException(msg)`| 412    |
| `NewPayloadTooLargeException(msg)`   | 413    |
| `NewUnsupportedMediaTypeException(msg)`| 415  |
| `NewImATeapotException(msg)`         | 418    |
| `NewMisdirectedException(msg)`       | 421    |
| `NewUnprocessableEntityException(msg)`| 422   |
| `NewTooManyRequestsException(msg)`   | 429    |
| `NewInternalServerError(msg)`        | 500    |
| `NewNotImplementedException(msg)`    | 501    |
| `NewBadGatewayException(msg)`        | 502    |
| `NewServiceUnavailableException(msg)`| 503    |
| `NewGatewayTimeoutException(msg)`    | 504    |
| `NewHttpVersionNotSupportedException(msg)`| 505 |

Custom exceptions:

```go
gonest.NewHTTPException(http.StatusTeapot, "I'm a teapot")
gonest.WrapHTTPException(http.StatusInternalServerError, "failed", err)
```

---

## Platform Adapter

GoNest ships with a built-in HTTP adapter based on `net/http` with a trie-based router.

```go
import "github.com/gonest/platform/stdhttp"

adapter := stdhttp.New()
```

Features:
- Trie-based path matching with `:param` parameters and `*` wildcards
- Custom not-found and method-not-allowed handlers
- Standard `http.Handler` interface for compatibility with any Go HTTP middleware

Implement the `platform.HTTPAdapter` interface to use a different router (Chi, Gorilla, etc.).

---

## Examples

The `example/` directory contains 26 example applications:

| #  | Example                | Description                                 |
|----|------------------------|---------------------------------------------|
| 01 | `cats-app`             | Basic REST API with CRUD                    |
| 02 | `websocket`            | WebSocket gateway                           |
| 03 | `microservice`         | TCP microservice                            |
| 04 | `auth-jwt`             | JWT authentication                          |
| 05 | `cache`                | Caching interceptor                         |
| 06 | `schedule`             | Job scheduling (cron, interval, timeout)    |
| 07 | `sse`                  | Server-Sent Events                          |
| 08 | `config`               | ConfigService with .env files               |
| 09 | `graphql`              | GraphQL engine with playground              |
| 10 | `versioning`           | API versioning strategies                   |
| 11 | `health`               | Health check indicators                     |
| 12 | `throttle`             | Rate limiting                               |
| 13 | `serializer`           | Response serialization with groups          |
| 14 | `sql-database`         | SQL database integration                    |
| 15 | `serve-static`         | Static file serving                         |
| 16 | `dynamic-modules`      | Dynamic module configuration                |
| 17 | `queues`               | In-memory job queues                        |
| 18 | `file-upload`          | File upload with validation                 |
| 19 | `event-emitter`        | Event pub/sub system                        |
| 20 | `swagger`              | OpenAPI / Swagger UI                        |
| 21 | `mvc`                  | Template rendering                          |
| 22 | `mongo`                | MongoDB integration                         |
| 23 | `grpc`                 | gRPC microservice transport                 |
| 24 | `graphql-federation`   | GraphQL federation                          |
| 25 | `context`              | Context usage patterns                      |
| -- | `fullstack-api`        | Complete boilerplate (users, articles, auth, files) |

Run any example:

```bash
cd example/01-cats-app
go run main.go
```

---

## Project Structure

```
github.com/gonest
|-- gonest.go                  # Application bootstrap & HTTP pipeline
|-- module.go                  # Module system
|-- container.go               # DI container
|-- provider.go                # Provider definitions
|-- scope.go                   # Lifetime scopes
|-- route.go                   # Route & RouteBuilder
|-- context.go                 # Request context
|-- errors.go                  # HTTP exceptions
|-- middleware.go               # Middleware interfaces
|-- guard.go                   # Guard interface
|-- interceptor.go             # Interceptor interface
|-- pipe.go                    # Pipe interface & built-in pipes
|-- filter.go                  # Exception filter interface
|-- validation.go              # ValidationPipe
|-- session.go                 # Session management
|-- sse.go                     # Server-Sent Events
|-- events.go                  # Event emitter
|-- throttle.go                # Rate limiting
|-- versioning.go              # API versioning
|-- serializer.go              # Response serialization
|-- fileupload.go              # File upload handling
|-- streamable_file.go         # File streaming
|-- rawbody.go                 # Raw body access
|-- host.go                    # Host/subdomain routing
|-- render.go / template.go    # Template rendering
|-- reflector.go               # Metadata reflection
|-- discovery.go               # Discovery service
|-- lazy_module.go             # Lazy module loading
|-- repl.go                    # Interactive REPL
|-- lifecycle.go               # Lifecycle hook interfaces
|-- forward_ref.go             # Circular dependency resolution
|-- configurable_module.go     # Configurable module builder
|-- application_context.go     # Non-HTTP application context
|-- platform/
|   |-- adapter.go             # HTTPAdapter interface
|   `-- stdhttp/               # Built-in net/http adapter
|-- config/                    # Configuration module
|-- cache/                     # Caching module
|-- schedule/                  # Scheduling module
|-- queue/                     # Queue module
|-- websocket/                 # WebSocket gateway
|-- microservice/              # Microservice transports
|   |-- tcp/                   # TCP transport
|   |-- grpc/                  # gRPC transport
|   |-- nats/                  # NATS transport
|   |-- redis/                 # Redis transport
|   |-- kafka/                 # Kafka transport
|   |-- rabbitmq/              # RabbitMQ transport
|   `-- mqtt/                  # MQTT transport
|-- graphql/                   # GraphQL engine
|-- swagger/                   # OpenAPI / Swagger
|-- database/                  # Repository interface & pagination
|   |-- sql/                   # SQL module
|   `-- mongo/                 # MongoDB module
|-- health/                    # Health checks
|-- testing/                   # Test utilities
`-- example/                   # Example applications
```

---

## License

See [LICENSE](LICENSE) for details.
