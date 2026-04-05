package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xfurai/gonest"
)

// ---------------------------------------------------------------------------
// Services
// ---------------------------------------------------------------------------

type helloService struct{}

func newHelloService() *helloService { return &helloService{} }

func (s *helloService) Greeting() string { return "Hello World!" }

type usersService struct{}

func newUsersService() *usersService { return &usersService{} }

func (s *usersService) FindAll() []map[string]any {
	return []map[string]any{
		{"id": 1, "name": "Alice"},
		{"id": 2, "name": "Bob"},
	}
}

// ---------------------------------------------------------------------------
// Controllers
// ---------------------------------------------------------------------------

type helloController struct {
	svc *helloService
}

func newHelloController(svc *helloService) *helloController {
	return &helloController{svc: svc}
}

func (c *helloController) Register(r gonest.Router) {
	r.Prefix("/hello")
	r.Get("", c.index)
	r.Get("/greeting", c.greeting)
	r.Get("/async", c.async)
	r.Get("/param/:name", c.param)
	r.Post("/echo", c.echo)
	r.Head("", c.head)
}

func (c *helloController) index(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": c.svc.Greeting()})
}

func (c *helloController) greeting(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, c.svc.Greeting())
}

func (c *helloController) async(ctx gonest.Context) error {
	// Simulates async handler (in Go everything is sync, but still tests the path)
	return ctx.JSON(http.StatusOK, map[string]string{"message": "async " + c.svc.Greeting()})
}

func (c *helloController) param(ctx gonest.Context) error {
	name := ctx.Param("name")
	return ctx.JSON(http.StatusOK, map[string]any{"name": name})
}

func (c *helloController) echo(ctx gonest.Context) error {
	var body map[string]any
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	return ctx.JSON(http.StatusCreated, body)
}

func (c *helloController) head(ctx gonest.Context) error {
	ctx.SetHeader("X-Custom", "head-value")
	return ctx.NoContent(http.StatusOK)
}

type usersController struct {
	svc *usersService
}

func newUsersController(svc *usersService) *usersController {
	return &usersController{svc: svc}
}

func (c *usersController) Register(r gonest.Router) {
	r.Prefix("/users")
	r.Get("", c.findAll)
}

func (c *usersController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.svc.FindAll())
}

// ---------------------------------------------------------------------------
// Module
// ---------------------------------------------------------------------------

func helloWorldModule() *gonest.Module {
	return gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHelloController, newUsersController},
		Providers:   []any{newHelloService, newUsersService},
	})
}

func createHelloWorldApp(t *testing.T) *gonest.Application {
	t.Helper()
	app := gonest.Create(helloWorldModule(), gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init: %v", err)
	}
	return app
}

// ---------------------------------------------------------------------------
// Tests: Basic HTTP
// ---------------------------------------------------------------------------

func TestHelloWorld_GetIndex(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "Hello World!" {
		t.Errorf("expected Hello World!, got %q", body["message"])
	}
}

func TestHelloWorld_GetGreeting(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello/greeting", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "Hello World!" {
		t.Errorf("expected Hello World!, got %q", w.Body.String())
	}
}

func TestHelloWorld_AsyncHandler(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello/async", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "async Hello World!" {
		t.Errorf("unexpected message: %q", body["message"])
	}
}

func TestHelloWorld_PathParam(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello/param/NestJS", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["name"] != "NestJS" {
		t.Errorf("expected NestJS, got %v", body["name"])
	}
}

func TestHelloWorld_PostEcho(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	payload := `{"foo":"bar"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/hello/echo", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["foo"] != "bar" {
		t.Errorf("expected bar, got %v", body["foo"])
	}
}

func TestHelloWorld_HeadRequest(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("HEAD", "/hello", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Custom") != "head-value" {
		t.Errorf("expected X-Custom header")
	}
}

func TestHelloWorld_NotFound(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/nonexistent", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHelloWorld_UsersRoute(t *testing.T) {
	app := createHelloWorldApp(t)
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/users", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var users []map[string]any
	json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

// ---------------------------------------------------------------------------
// Tests: Guards
// ---------------------------------------------------------------------------

type authGuard struct{}

func (g *authGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	token := ctx.Header("Authorization")
	if token == "" {
		return false, gonest.NewUnauthorizedException("Unauthorized")
	}
	return true, nil
}

type roleGuard struct{}

func (g *roleGuard) CanActivate(ctx gonest.ExecutionContext) (bool, error) {
	role, ok := gonest.GetMetadata[string](ctx, "role")
	if !ok {
		return true, nil
	}
	if ctx.Header("X-Role") != role {
		return false, gonest.NewForbiddenException("Forbidden resource")
	}
	return true, nil
}

type guardedController struct{}

func newGuardedController() *guardedController { return &guardedController{} }

func (c *guardedController) Register(r gonest.Router) {
	r.Prefix("/guarded")
	r.Get("/public", c.public)
	r.Get("/admin", c.admin).SetMetadata("role", "admin")
}

func (c *guardedController) public(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (c *guardedController) admin(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"status": "admin"})
}

func TestHelloWorld_Guard_DenyWithoutToken(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newGuardedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalGuards(&authGuard{})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/guarded/public", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHelloWorld_Guard_AllowWithToken(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newGuardedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalGuards(&authGuard{})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/guarded/public", nil)
	req.Header.Set("Authorization", "Bearer token")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHelloWorld_Guard_RoleMetadata(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newGuardedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalGuards(&roleGuard{})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Without correct role
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/guarded/admin", nil)
	req.Header.Set("X-Role", "user")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	// With correct role
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/guarded/admin", nil)
	req.Header.Set("X-Role", "admin")
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHelloWorld_Guard_NoConstraintAllows(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newGuardedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalGuards(&roleGuard{}) // role guard, but /public has no role metadata
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/guarded/public", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Interceptors
// ---------------------------------------------------------------------------

type loggingInterceptor struct {
	calls *int
}

func (i *loggingInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
	*i.calls++
	ctx.SetHeader("X-Intercepted", "true")
	return next.Handle()
}

type transformInterceptor struct{}

func (i *transformInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
	result, err := next.Handle()
	if err != nil {
		return nil, err
	}
	ctx.SetHeader("X-Transformed", "true")
	return result, nil
}

func TestHelloWorld_Interceptor_SetsHeader(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHelloController},
		Providers:   []any{newHelloService},
	})
	calls := 0
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalInterceptors(&loggingInterceptor{calls: &calls})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Intercepted") != "true" {
		t.Error("expected X-Intercepted header")
	}
	if calls != 1 {
		t.Errorf("expected 1 interceptor call, got %d", calls)
	}
}

func TestHelloWorld_Interceptor_Ordering(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHelloController},
		Providers:   []any{newHelloService},
	})
	calls := 0
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalInterceptors(
		&loggingInterceptor{calls: &calls},
		&transformInterceptor{},
	)
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("X-Intercepted") != "true" {
		t.Error("expected X-Intercepted header")
	}
	if w.Header().Get("X-Transformed") != "true" {
		t.Error("expected X-Transformed header")
	}
}

func TestHelloWorld_Interceptor_RouteLevel(t *testing.T) {
	icCtrl := &interceptorRouteController{}
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *interceptorRouteController { return icCtrl }},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Route with interceptor
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ic/with", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Header().Get("X-Transformed") != "true" {
		t.Error("expected X-Transformed on intercepted route")
	}

	// Route without interceptor
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/ic/without", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Header().Get("X-Transformed") != "" {
		t.Error("unexpected X-Transformed on non-intercepted route")
	}
}

type interceptorRouteController struct{}

func (c *interceptorRouteController) Register(r gonest.Router) {
	r.Prefix("/ic")
	r.Get("/with", c.handler).Interceptors(&transformInterceptor{})
	r.Get("/without", c.handler)
}

func (c *interceptorRouteController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"ok": "true"})
}

// ---------------------------------------------------------------------------
// Tests: Middleware
// ---------------------------------------------------------------------------

type orderMiddleware struct {
	order *[]string
	name  string
}

func (m *orderMiddleware) Use(ctx gonest.Context, next gonest.NextFunc) error {
	*m.order = append(*m.order, m.name+":before")
	err := next()
	*m.order = append(*m.order, m.name+":after")
	return err
}

func TestHelloWorld_Middleware_ExecutionOrder(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHelloController},
		Providers:   []any{newHelloService},
	})
	var order []string
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(
		&orderMiddleware{order: &order, name: "mw1"},
		&orderMiddleware{order: &order, name: "mw2"},
		&orderMiddleware{order: &order, name: "mw3"},
	)
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, req)

	expected := []string{
		"mw1:before", "mw2:before", "mw3:before",
		"mw3:after", "mw2:after", "mw1:after",
	}
	if len(order) != len(expected) {
		t.Fatalf("expected %d entries, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d]: expected %q, got %q", i, v, order[i])
		}
	}
}

func TestHelloWorld_Middleware_ShortCircuit(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHelloController},
		Providers:   []any{newHelloService},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.MiddlewareFunc(func(ctx gonest.Context, next gonest.NextFunc) error {
		return ctx.String(http.StatusServiceUnavailable, "maintenance")
	}))
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
	if w.Body.String() != "maintenance" {
		t.Errorf("expected maintenance, got %q", w.Body.String())
	}
}

func TestHelloWorld_Middleware_FuncAdapter(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHelloController},
		Providers:   []any{newHelloService},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalMiddleware(gonest.MiddlewareFunc(func(ctx gonest.Context, next gonest.NextFunc) error {
		ctx.SetHeader("X-MW", "func")
		return next()
	}))
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("X-MW") != "func" {
		t.Errorf("expected X-MW=func header")
	}
}

// ---------------------------------------------------------------------------
// Tests: Pipes
// ---------------------------------------------------------------------------

type pipedController struct{}

func newPipedController() *pipedController { return &pipedController{} }

func (c *pipedController) Register(r gonest.Router) {
	r.Prefix("/piped")
	r.Get("/int/:id", c.intParam).Pipes(gonest.NewParseIntPipe("id"))
	r.Get("/bool/:flag", c.boolParam).Pipes(gonest.NewParseBoolPipe("flag"))
	r.Get("/uuid/:id", c.uuidParam).Pipes(gonest.NewParseUUIDPipe("id"))
	r.Get("/default/:val", c.defaultParam).Pipes(gonest.NewDefaultValuePipe("val", "fallback"))
	r.Get("/array/:items", c.arrayParam).Pipes(gonest.NewParseArrayPipe("items"))
}

func (c *pipedController) intParam(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"id": ctx.Param("id")})
}

func (c *pipedController) boolParam(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"flag": ctx.Param("flag")})
}

func (c *pipedController) uuidParam(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"id": ctx.Param("id")})
}

func (c *pipedController) defaultParam(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"val": ctx.Param("val")})
}

func (c *pipedController) arrayParam(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"items": ctx.Param("items")})
}

func TestHelloWorld_Pipe_ParseInt(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newPipedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Valid int
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/piped/int/42", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["id"] != float64(42) {
		t.Errorf("expected 42, got %v", body["id"])
	}

	// Invalid int
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/piped/int/abc", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHelloWorld_Pipe_ParseBool(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newPipedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/piped/bool/true", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["flag"] != true {
		t.Errorf("expected true, got %v", body["flag"])
	}
}

func TestHelloWorld_Pipe_ParseUUID(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newPipedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Valid UUID
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/piped/uuid/550e8400-e29b-41d4-a716-446655440000", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Invalid UUID
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/piped/uuid/not-a-uuid", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHelloWorld_Pipe_GlobalPipe(t *testing.T) {
	ctrl := &globalPipeController{}
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{func() *globalPipeController { return ctrl }},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalPipes(gonest.NewParseIntPipe("id"))
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/gp/10", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["id"] != float64(10) {
		t.Errorf("expected 10, got %v", body["id"])
	}
}

type globalPipeController struct{}

func (c *globalPipeController) Register(r gonest.Router) {
	r.Get("/gp/:id", c.handler)
}

func (c *globalPipeController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"id": ctx.Param("id")})
}

// ---------------------------------------------------------------------------
// Tests: Exception Filters
// ---------------------------------------------------------------------------

type customExceptionFilter struct{}

func (f *customExceptionFilter) Catch(err error, host gonest.ArgumentsHost) error {
	httpCtx := host.SwitchToHTTP()
	resp := httpCtx.Response()
	if httpErr, ok := err.(*gonest.HTTPException); ok {
		resp.Header().Set("Content-Type", "application/json; charset=utf-8")
		resp.WriteHeader(httpErr.StatusCode())
		json.NewEncoder(resp).Encode(map[string]any{
			"error":      httpErr.Error(),
			"statusCode": httpErr.StatusCode(),
			"custom":     true,
		})
		return nil
	}
	return err
}

type exceptionController struct{}

func newExceptionController() *exceptionController { return &exceptionController{} }

func (c *exceptionController) Register(r gonest.Router) {
	r.Prefix("/exception")
	r.Get("/bad-request", c.badRequest)
	r.Get("/not-found", c.notFound)
	r.Get("/forbidden", c.forbidden)
	r.Get("/generic", c.generic)
}

func (c *exceptionController) badRequest(ctx gonest.Context) error {
	return gonest.NewBadRequestException("invalid input")
}

func (c *exceptionController) notFound(ctx gonest.Context) error {
	return gonest.NewNotFoundException("resource not found")
}

func (c *exceptionController) forbidden(ctx gonest.Context) error {
	return gonest.NewForbiddenException("access denied")
}

func (c *exceptionController) generic(ctx gonest.Context) error {
	return gonest.NewInternalServerError("something broke")
}

func TestHelloWorld_ExceptionFilter_Default(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newExceptionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	tests := []struct {
		path   string
		status int
		msg    string
	}{
		{"/exception/bad-request", 400, "invalid input"},
		{"/exception/not-found", 404, "resource not found"},
		{"/exception/forbidden", 403, "access denied"},
		{"/exception/generic", 500, "something broke"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", tt.path, nil)
		app.Handler().ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("%s: expected %d, got %d", tt.path, tt.status, w.Code)
			continue
		}
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		if body["message"] != tt.msg {
			t.Errorf("%s: expected message %q, got %v", tt.path, tt.msg, body["message"])
		}
		if body["statusCode"] != float64(tt.status) {
			t.Errorf("%s: expected statusCode %d, got %v", tt.path, tt.status, body["statusCode"])
		}
	}
}

func TestHelloWorld_ExceptionFilter_Custom(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newExceptionController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalFilters(&customExceptionFilter{})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/exception/bad-request", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["custom"] != true {
		t.Errorf("expected custom filter to be applied")
	}
}

// ---------------------------------------------------------------------------
// Tests: Response Headers via Route Builder
// ---------------------------------------------------------------------------

type headerController struct{}

func newHeaderController() *headerController { return &headerController{} }

func (c *headerController) Register(r gonest.Router) {
	r.Get("/with-header", c.handler).Header("X-Custom-Header", "custom-value")
}

func (c *headerController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"ok": "true"})
}

func TestHelloWorld_RouteHeader(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newHeaderController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/with-header", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header=custom-value")
	}
}

// ---------------------------------------------------------------------------
// Tests: Redirect
// ---------------------------------------------------------------------------

type redirectController struct{}

func newRedirectController() *redirectController { return &redirectController{} }

func (c *redirectController) Register(r gonest.Router) {
	r.Get("/old", c.handler).Redirect("/new", http.StatusMovedPermanently)
}

func (c *redirectController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, nil) // should not be reached
}

func TestHelloWorld_Redirect(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newRedirectController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/old", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/new" {
		t.Errorf("expected Location=/new, got %q", w.Header().Get("Location"))
	}
}

// ---------------------------------------------------------------------------
// Tests: Router Module (nested modules with routes)
// ---------------------------------------------------------------------------

type catService struct{}

func newCatService() *catService { return &catService{} }

func (s *catService) FindAll() []map[string]string {
	return []map[string]string{{"name": "Whiskers"}}
}

type catController struct {
	svc *catService
}

func newCatController(svc *catService) *catController {
	return &catController{svc: svc}
}

func (c *catController) Register(r gonest.Router) {
	r.Prefix("/cats")
	r.Get("", c.findAll)
}

func (c *catController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.svc.FindAll())
}

func TestHelloWorld_RouterModule(t *testing.T) {
	catModule := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newCatController},
		Providers:   []any{newCatService},
	})
	rootModule := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{catModule},
		Controllers: []any{newHelloController},
		Providers:   []any{newHelloService},
	})

	app := gonest.Create(rootModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Root route
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/hello", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /hello, got %d", w.Code)
	}

	// Imported module route
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/cats", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /cats, got %d", w.Code)
	}

	var cats []map[string]string
	json.Unmarshal(w.Body.Bytes(), &cats)
	if len(cats) != 1 || cats[0]["name"] != "Whiskers" {
		t.Errorf("unexpected cats response: %v", cats)
	}
}

// ---------------------------------------------------------------------------
// Tests: MiddlewareConsumer (forRoutes / exclude)
// Mirror: original/integration/hello-world/e2e/middleware.spec.ts
// Mirror: original/integration/hello-world/e2e/exclude-middleware.spec.ts
// ---------------------------------------------------------------------------

type mwConfigService struct {
	applied *[]string
}

func (s *mwConfigService) Configure(consumer gonest.MiddlewareConsumer) {
	consumer.Apply(gonest.MiddlewareFunc(func(ctx gonest.Context, next gonest.NextFunc) error {
		*s.applied = append(*s.applied, ctx.Path())
		ctx.SetHeader("X-Middleware", "applied")
		return next()
	})).ForRoutes("/mw/*")
}

type mwConfigController struct{}

func newMwConfigController() *mwConfigController { return &mwConfigController{} }

func (c *mwConfigController) Register(r gonest.Router) {
	r.Get("/mw/hello", c.hello)
	r.Get("/mw/test", c.test)
	r.Get("/other", c.other)
}

func (c *mwConfigController) hello(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"route": "hello"})
}

func (c *mwConfigController) test(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"route": "test"})
}

func (c *mwConfigController) other(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"route": "other"})
}

func TestHelloWorld_MiddlewareConsumer_ForRoutes(t *testing.T) {
	var applied []string
	configSvc := &mwConfigService{applied: &applied}

	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newMwConfigController},
		Providers:   []any{gonest.ProvideValue[*mwConfigService](configSvc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Route matching forRoutes("/mw/*") should have middleware
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/mw/hello", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("X-Middleware") != "applied" {
		t.Error("expected middleware on /mw/hello")
	}

	// Route NOT matching forRoutes should NOT have middleware
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/other", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("X-Middleware") != "" {
		t.Error("middleware should NOT apply to /other")
	}
}

type excludeMwConfigService struct{}

func (s *excludeMwConfigService) Configure(consumer gonest.MiddlewareConsumer) {
	consumer.Apply(gonest.MiddlewareFunc(func(ctx gonest.Context, next gonest.NextFunc) error {
		ctx.SetHeader("X-MW-Excluded", "applied")
		return next()
	})).Exclude("/excl/skip").ForRoutes("/excl/*")
}

type excludeController struct{}

func newExcludeController() *excludeController { return &excludeController{} }

func (c *excludeController) Register(r gonest.Router) {
	r.Prefix("/excl")
	r.Get("/keep", c.keep)
	r.Get("/skip", c.skip)
	r.Get("/also-keep", c.alsoKeep)
}

func (c *excludeController) keep(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "keep")
}

func (c *excludeController) skip(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "skip")
}

func (c *excludeController) alsoKeep(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "also-keep")
}

func TestHelloWorld_MiddlewareConsumer_Exclude(t *testing.T) {
	configSvc := &excludeMwConfigService{}
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newExcludeController},
		Providers:   []any{gonest.ProvideValue[*excludeMwConfigService](configSvc)},
	})

	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Included route should have middleware
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/excl/keep", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Header().Get("X-MW-Excluded") != "applied" {
		t.Error("expected middleware on /excl/keep")
	}

	// Excluded route should NOT have middleware
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/excl/skip", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Header().Get("X-MW-Excluded") != "" {
		t.Error("middleware should be excluded from /excl/skip")
	}

	// Another included route
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/excl/also-keep", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Header().Get("X-MW-Excluded") != "applied" {
		t.Error("expected middleware on /excl/also-keep")
	}
}

// ---------------------------------------------------------------------------
// Tests: Nested Router Module (parent/child prefixes)
// Mirror: original/integration/hello-world/e2e/router-module.spec.ts
// ---------------------------------------------------------------------------

type parentController struct{}

func newParentController() *parentController { return &parentController{} }

func (c *parentController) Register(r gonest.Router) {
	r.Prefix("/parent")
	r.Get("/action", c.handler)
}

func (c *parentController) handler(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "ParentController")
}

type childController struct{}

func newChildController() *childController { return &childController{} }

func (c *childController) Register(r gonest.Router) {
	r.Prefix("/parent/child")
	r.Get("/action", c.handler)
}

func (c *childController) handler(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "ChildController")
}

func TestHelloWorld_NestedRouterModules(t *testing.T) {
	childMod := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newChildController},
	})
	parentMod := gonest.NewModule(gonest.ModuleOptions{
		Imports:     []*gonest.Module{childMod},
		Controllers: []any{newParentController},
	})

	app := gonest.Create(parentMod, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Parent route
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/parent/action", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /parent/action, got %d", w.Code)
	}
	if w.Body.String() != "ParentController" {
		t.Errorf("expected ParentController, got %q", w.Body.String())
	}

	// Child route
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/parent/child/action", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on /parent/child/action, got %d", w.Code)
	}
	if w.Body.String() != "ChildController" {
		t.Errorf("expected ChildController, got %q", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tests: Controller-level guards and interceptors
// Mirror: original/integration/hello-world/e2e/guards.spec.ts (controller guards)
// ---------------------------------------------------------------------------

type controllerGuardedController struct{}

func newControllerGuardedController() *controllerGuardedController {
	return &controllerGuardedController{}
}

func (c *controllerGuardedController) Register(r gonest.Router) {
	r.Prefix("/ctrl-guarded")
	r.UseGuards(&authGuard{})
	r.Get("/one", c.one)
	r.Get("/two", c.two)
}

func (c *controllerGuardedController) one(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "one")
}

func (c *controllerGuardedController) two(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "two")
}

func TestHelloWorld_ControllerLevelGuard(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newControllerGuardedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Without token — both routes blocked
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ctrl-guarded/one", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/ctrl-guarded/two", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	// With token — both routes allowed
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/ctrl-guarded/one", nil)
	req.Header.Set("Authorization", "Bearer x")
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Controller-level interceptors
// ---------------------------------------------------------------------------

type controllerInterceptedController struct{}

func newControllerInterceptedController() *controllerInterceptedController {
	return &controllerInterceptedController{}
}

func (c *controllerInterceptedController) Register(r gonest.Router) {
	r.Prefix("/ctrl-ic")
	r.UseInterceptors(&transformInterceptor{})
	r.Get("/a", c.handler)
	r.Get("/b", c.handler)
}

func (c *controllerInterceptedController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"ok": "true"})
}

func TestHelloWorld_ControllerLevelInterceptor(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newControllerInterceptedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	for _, path := range []string{"/ctrl-ic/a", "/ctrl-ic/b"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		app.Handler().ServeHTTP(w, req)
		if w.Header().Get("X-Transformed") != "true" {
			t.Errorf("%s: expected X-Transformed header", path)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Controller-level pipes
// ---------------------------------------------------------------------------

type controllerPipedController struct{}

func newControllerPipedController() *controllerPipedController {
	return &controllerPipedController{}
}

func (c *controllerPipedController) Register(r gonest.Router) {
	r.Prefix("/ctrl-pipe")
	r.UsePipes(gonest.NewParseIntPipe("id"))
	r.Get("/:id", c.handler)
}

func (c *controllerPipedController) handler(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{"id": ctx.Param("id")})
}

func TestHelloWorld_ControllerLevelPipe(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newControllerPipedController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	// Valid int
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ctrl-pipe/5", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["id"] != float64(5) {
		t.Errorf("expected 5, got %v", body["id"])
	}

	// Invalid int
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/ctrl-pipe/abc", nil)
	app.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Tests: Controller-level exception filters
// ---------------------------------------------------------------------------

type controllerFilteredController struct{}

func newControllerFilteredController() *controllerFilteredController {
	return &controllerFilteredController{}
}

func (c *controllerFilteredController) Register(r gonest.Router) {
	r.Prefix("/ctrl-filter")
	r.Get("/error", c.errorRoute)
}

func (c *controllerFilteredController) errorRoute(ctx gonest.Context) error {
	return gonest.NewBadRequestException("controller filter test")
}

func TestHelloWorld_ControllerLevelFilter(t *testing.T) {
	// Use a dedicated filter that sets a unique header so we can confirm it ran
	filter := gonest.ExceptionFilterFunc(func(err error, host gonest.ArgumentsHost) error {
		if httpErr, ok := err.(*gonest.HTTPException); ok {
			httpCtx := host.SwitchToHTTP()
			resp := httpCtx.Response()
			resp.Header().Set("Content-Type", "application/json; charset=utf-8")
			resp.Header().Set("X-Custom-Filter", "true")
			resp.WriteHeader(httpErr.StatusCode())
			json.NewEncoder(resp).Encode(map[string]any{
				"error":      httpErr.Error(),
				"statusCode": httpErr.StatusCode(),
				"filtered":   true,
			})
			return nil
		}
		return err
	})

	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newControllerFilteredController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	app.UseGlobalFilters(filter)
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ctrl-filter/error", nil)
	app.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if w.Header().Get("X-Custom-Filter") != "true" {
		t.Errorf("expected X-Custom-Filter header, response body: %s", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple HTTP methods on same path
// Mirror: original/integration/hello-world/e2e/express-instance.spec.ts
// ---------------------------------------------------------------------------

type multiMethodController struct{}

func newMultiMethodController() *multiMethodController { return &multiMethodController{} }

func (c *multiMethodController) Register(r gonest.Router) {
	r.Get("/resource", c.get)
	r.Post("/resource", c.post)
	r.Put("/resource", c.put)
	r.Delete("/resource", c.del)
	r.Patch("/resource", c.patch)
}

func (c *multiMethodController) get(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "GET")
}

func (c *multiMethodController) post(ctx gonest.Context) error {
	return ctx.String(http.StatusCreated, "POST")
}

func (c *multiMethodController) put(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "PUT")
}

func (c *multiMethodController) del(ctx gonest.Context) error {
	return ctx.NoContent(http.StatusNoContent)
}

func (c *multiMethodController) patch(ctx gonest.Context) error {
	return ctx.String(http.StatusOK, "PATCH")
}

func TestHelloWorld_MultipleHTTPMethods(t *testing.T) {
	module := gonest.NewModule(gonest.ModuleOptions{
		Controllers: []any{newMultiMethodController},
	})
	app := gonest.Create(module, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatal(err)
	}
	defer app.Close()

	tests := []struct {
		method string
		status int
		body   string
	}{
		{"GET", 200, "GET"},
		{"POST", 201, "POST"},
		{"PUT", 200, "PUT"},
		{"DELETE", 204, ""},
		{"PATCH", 200, "PATCH"},
	}

	for _, tt := range tests {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, "/resource", nil)
		app.Handler().ServeHTTP(w, req)
		if w.Code != tt.status {
			t.Errorf("%s: expected %d, got %d", tt.method, tt.status, w.Code)
		}
		if tt.body != "" && w.Body.String() != tt.body {
			t.Errorf("%s: expected %q, got %q", tt.method, tt.body, w.Body.String())
		}
	}
}
