package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gonest"
)

// --- DTOs ---

// Item represents a simple resource.
type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// --- Guard: AuthGuard ---

// AuthGuard demonstrates reading metadata from the ExecutionContext to
// decide whether a request should be allowed. It checks for a "public"
// metadata flag; protected routes require an Authorization header.
type AuthGuard struct{}

func (g *AuthGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	// Check if the route is marked as public via SetMetadata("public", true).
	isPublic, ok := gonest.GetMetadata[bool](ctx, "public")
	if ok && isPublic {
		log.Printf("[AuthGuard] Route is public, allowing access")
		return true, nil
	}

	// For protected routes, require an Authorization header.
	authHeader := ctx.Header("Authorization")
	if authHeader == "" {
		log.Printf("[AuthGuard] No Authorization header, denying access")
		return false, gonest.NewUnauthorizedException("missing Authorization header")
	}

	// Demonstrate SwitchToHTTP() to access the raw request.
	httpCtx := ctx.SwitchToHTTP()
	req := httpCtx.Request()
	log.Printf("[AuthGuard] Authorized request: %s %s (User-Agent: %s)",
		req.Method, req.URL.Path, req.Header.Get("User-Agent"))

	return true, nil
}

// --- Guard: RolesGuard ---

// RolesGuard demonstrates reading a "roles" metadata array from the
// ExecutionContext. Only users whose X-User-Role header matches one
// of the allowed roles may proceed.
type RolesGuard struct{}

func (g *RolesGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	// Read the "roles" metadata set on the route via SetMetadata("roles", ...).
	roles, ok := gonest.GetMetadata[[]string](ctx, "roles")
	if !ok || len(roles) == 0 {
		// No roles restriction on this route.
		log.Printf("[RolesGuard] No role restriction, allowing access")
		return true, nil
	}

	userRole := ctx.Header("X-User-Role")
	if userRole == "" {
		log.Printf("[RolesGuard] No X-User-Role header")
		return false, gonest.NewForbiddenException("no role provided")
	}

	for _, r := range roles {
		if r == userRole {
			log.Printf("[RolesGuard] User role %q matches allowed roles %v", userRole, roles)
			return true, nil
		}
	}

	log.Printf("[RolesGuard] User role %q not in allowed roles %v", userRole, roles)
	return false, gonest.NewForbiddenException(
		fmt.Sprintf("role %q is not authorized; required: %s", userRole, strings.Join(roles, ", ")),
	)
}

// --- Interceptor: LoggingInterceptor ---

// LoggingInterceptor demonstrates inspecting the ExecutionContext to log
// details about the handler, controller, metadata, and execution time.
type LoggingInterceptor struct{}

func (i *LoggingInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
	start := time.Now()

	// Inspect the execution context.
	handler := ctx.GetHandler()
	class := ctx.GetClass()
	contextType := ctx.GetType()

	log.Printf("[LoggingInterceptor] Before handler")
	log.Printf("  Context type : %s", contextType)
	log.Printf("  Handler      : %v", handler)
	log.Printf("  Controller   : %T", class)
	log.Printf("  Method       : %s %s", ctx.Method(), ctx.Path())

	// Read all known metadata keys to demonstrate GetMetadata.
	if summary, ok := ctx.GetMetadata("summary"); ok {
		log.Printf("  Summary      : %v", summary)
	}
	if tags, ok := ctx.GetMetadata("tags"); ok {
		log.Printf("  Tags         : %v", tags)
	}
	if roles, ok := ctx.GetMetadata("roles"); ok {
		log.Printf("  Roles        : %v", roles)
	}
	if isPublic, ok := ctx.GetMetadata("public"); ok {
		log.Printf("  Public       : %v", isPublic)
	}

	// Demonstrate SwitchToHTTP() in the interceptor.
	httpCtx := ctx.SwitchToHTTP()
	req := httpCtx.Request()
	log.Printf("  Remote addr  : %s", req.RemoteAddr)
	log.Printf("  Content-Type : %s", req.Header.Get("Content-Type"))

	// Execute the handler.
	result, err := next.Handle()

	elapsed := time.Since(start)
	if err != nil {
		log.Printf("[LoggingInterceptor] After handler (error: %v, took %v)", err, elapsed)
	} else {
		log.Printf("[LoggingInterceptor] After handler (success, took %v)", elapsed)
	}

	return result, err
}

// --- Interceptor: TransformInterceptor ---

// TransformInterceptor demonstrates wrapping the response in a standard
// envelope, adding timing information via the execution context.
type TransformInterceptor struct{}

func (i *TransformInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
	start := time.Now()
	result, err := next.Handle()
	elapsed := time.Since(start)

	// Add timing header using the context.
	ctx.SetHeader("X-Response-Time", elapsed.String())
	ctx.SetHeader("X-Context-Type", ctx.GetType())

	return result, err
}

// --- Controller ---

// ItemsController demonstrates using execution context features in guards
// and interceptors through route metadata.
type ItemsController struct{}

// NewItemsController creates a new ItemsController.
func NewItemsController() *ItemsController {
	return &ItemsController{}
}

// Register defines routes with metadata for guards and interceptors to consume.
func (c *ItemsController) Register(r gonest.Router) {
	r.Prefix("/items")

	// Apply the logging and transform interceptors to all routes in this controller.
	r.UseInterceptors(&LoggingInterceptor{}, &TransformInterceptor{})

	// GET /items — Public route, no auth required.
	// The AuthGuard reads "public" metadata and skips auth.
	r.Get("/", c.findAll).
		SetMetadata("public", true).
		SetMetadata("summary", "List all items").
		SetMetadata("tags", []string{"items"}).
		Guards(&AuthGuard{})

	// GET /items/:id — Protected route, requires Authorization header.
	// The AuthGuard sees no "public" metadata and enforces auth.
	r.Get("/:id", c.findOne).
		SetMetadata("summary", "Get item by ID").
		SetMetadata("tags", []string{"items"}).
		Pipes(gonest.NewParseIntPipe("id")).
		Guards(&AuthGuard{})

	// POST /items — Admin-only route.
	// Both AuthGuard and RolesGuard inspect metadata.
	r.Post("/", c.create).
		SetMetadata("summary", "Create a new item").
		SetMetadata("tags", []string{"items"}).
		SetMetadata("roles", []string{"admin"}).
		Guards(&AuthGuard{}, &RolesGuard{}).
		HttpCode(http.StatusCreated)

	// DELETE /items/:id — Admin or moderator route.
	// RolesGuard reads the "roles" metadata to enforce access control.
	r.Delete("/:id", c.remove).
		SetMetadata("summary", "Delete an item").
		SetMetadata("tags", []string{"items"}).
		SetMetadata("roles", []string{"admin", "moderator"}).
		Pipes(gonest.NewParseIntPipe("id")).
		Guards(&AuthGuard{}, &RolesGuard{})
}

// Sample data
var items = []Item{
	{ID: 1, Name: "Keyboard"},
	{ID: 2, Name: "Mouse"},
	{ID: 3, Name: "Monitor"},
}

func (c *ItemsController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, items)
}

func (c *ItemsController) findOne(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	for _, item := range items {
		if item.ID == id {
			return ctx.JSON(http.StatusOK, item)
		}
	}
	return gonest.NewNotFoundException(fmt.Sprintf("item #%d not found", id))
}

func (c *ItemsController) create(ctx gonest.Context) error {
	var item Item
	if err := ctx.Bind(&item); err != nil {
		return err
	}
	item.ID = len(items) + 1
	items = append(items, item)
	return ctx.JSON(http.StatusCreated, item)
}

func (c *ItemsController) remove(ctx gonest.Context) error {
	id := ctx.Param("id").(int)
	for i, item := range items {
		if item.ID == id {
			items = append(items[:i], items[i+1:]...)
			return ctx.NoContent(http.StatusNoContent)
		}
	}
	return gonest.NewNotFoundException(fmt.Sprintf("item #%d not found", id))
}

// --- Middleware ---

// RequestIDMiddleware demonstrates a middleware that sets a value in the
// request-scoped store, which guards and interceptors can later read via
// the execution context.
type RequestIDMiddleware struct {
	counter int
}

func (m *RequestIDMiddleware) Use(ctx gonest.Context, next gonest.NextFunc) error {
	m.counter++
	requestID := fmt.Sprintf("req-%d-%d", time.Now().UnixMilli(), m.counter)
	ctx.Set("requestId", requestID)
	ctx.SetHeader("X-Request-ID", requestID)
	log.Printf("[RequestIDMiddleware] Assigned request ID: %s", requestID)
	return next()
}

// --- Module ---

// ItemsModule bundles the items controller.
var ItemsModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewItemsController},
})

// AppModule is the root application module.
var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Imports: []*gonest.Module{ItemsModule},
})

// --- Bootstrap ---

func main() {
	app := gonest.Create(AppModule)
	app.UseGlobalMiddleware(&RequestIDMiddleware{})
	app.EnableCors()

	log.Println("ExecutionContext example running at http://localhost:3000")
	log.Println("")
	log.Println("Endpoints:")
	log.Println("  GET    /items      — Public (no auth needed)")
	log.Println("  GET    /items/:id  — Protected (needs Authorization header)")
	log.Println("  POST   /items      — Admin only (needs Authorization + X-User-Role: admin)")
	log.Println("  DELETE /items/:id  — Admin/moderator (needs Authorization + X-User-Role: admin|moderator)")
	log.Println("")
	log.Println("Try:")
	log.Println("  curl http://localhost:3000/items")
	log.Println("  curl http://localhost:3000/items/1 -H 'Authorization: Bearer token'")
	log.Println("  curl -X POST http://localhost:3000/items -H 'Authorization: Bearer token' -H 'X-User-Role: admin' -H 'Content-Type: application/json' -d '{\"name\":\"Tablet\"}'")
	log.Fatal(app.Listen(":3000"))
}
