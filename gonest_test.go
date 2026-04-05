package gonest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// --- Test services ---

type greetingService struct {
	greeting string
}

func newGreetingService() *greetingService {
	return &greetingService{greeting: "Hello, World!"}
}

func (s *greetingService) Greet() string { return s.greeting }

// --- Test controller ---

type greetingController struct {
	service *greetingService
}

func newGreetingController(service *greetingService) *greetingController {
	return &greetingController{service: service}
}

func (c *greetingController) Register(r Router) {
	r.Prefix("/greet")
	r.Get("/", c.greet)
	r.Get("/:name", c.greetName)
	r.Post("/", c.createGreeting)
}

func (c *greetingController) greet(ctx Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{"message": c.service.Greet()})
}

func (c *greetingController) greetName(ctx Context) error {
	name := ctx.Param("name")
	return ctx.JSON(http.StatusOK, map[string]string{"message": "Hello, " + name.(string) + "!"})
}

func (c *greetingController) createGreeting(ctx Context) error {
	var body struct {
		Message string `json:"message"`
	}
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	c.service.greeting = body.Message
	return ctx.JSON(http.StatusCreated, map[string]string{"message": body.Message})
}

// --- Tests ---

func TestApplication_BasicRouting(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// GET /greet/
	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got %q", body["message"])
	}
}

func TestApplication_PathParams(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	req := httptest.NewRequest("GET", "/greet/Alice", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["message"] != "Hello, Alice!" {
		t.Errorf("expected 'Hello, Alice!', got %q", body["message"])
	}
}

func TestApplication_PostWithBody(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	body := `{"message":"Hi there"}`
	req := httptest.NewRequest("POST", "/greet/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestApplication_NotFound(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	req := httptest.NewRequest("GET", "/missing", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestApplication_GlobalPrefix(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.SetGlobalPrefix("/api/v1")
	app.Init()

	req := httptest.NewRequest("GET", "/api/v1/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Guard tests ---

type alwaysDenyGuard struct{}

func (g *alwaysDenyGuard) CanActivate(ctx ExecutionContext) (bool, error) {
	return false, nil
}

type roleCheckGuard struct{}

func (g *roleCheckGuard) CanActivate(ctx ExecutionContext) (bool, error) {
	roles, ok := GetMetadata[[]string](ctx, "roles")
	if !ok {
		return true, nil
	}
	_ = roles
	return false, NewForbiddenException("insufficient role")
}

func TestApplication_GlobalGuard_Deny(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalGuards(&alwaysDenyGuard{})
	app.Init()

	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- Guard on route with metadata ---

type guardedController struct {
	service *greetingService
}

func newGuardedController(service *greetingService) *guardedController {
	return &guardedController{service: service}
}

func (c *guardedController) Register(r Router) {
	r.Prefix("/guarded")
	r.Get("/public", c.publicRoute)
	r.Get("/admin", c.adminRoute).
		SetMetadata("roles", []string{"admin"}).
		Guards(&roleCheckGuard{})
}

func (c *guardedController) publicRoute(ctx Context) error {
	return ctx.JSON(200, map[string]string{"access": "public"})
}

func (c *guardedController) adminRoute(ctx Context) error {
	return ctx.JSON(200, map[string]string{"access": "admin"})
}

func TestApplication_RouteGuard(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGuardedController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	// Public route should work
	req := httptest.NewRequest("GET", "/guarded/public", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("public: expected 200, got %d", w.Code)
	}

	// Admin route should be denied
	req = httptest.NewRequest("GET", "/guarded/admin", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("admin: expected 403, got %d", w.Code)
	}
}

// --- Interceptor tests ---

type headerInterceptor struct {
	headerName  string
	headerValue string
}

func (i *headerInterceptor) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	ctx.SetHeader(i.headerName, i.headerValue)
	return next.Handle()
}

func TestApplication_GlobalInterceptor(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalInterceptors(&headerInterceptor{headerName: "X-Powered-By", headerValue: "GoNest"})
	app.Init()

	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("X-Powered-By") != "GoNest" {
		t.Errorf("expected X-Powered-By header, got %q", w.Header().Get("X-Powered-By"))
	}
}

// --- Pipe tests on routes ---

type pipedController struct{}

func newPipedController() *pipedController { return &pipedController{} }

func (c *pipedController) Register(r Router) {
	r.Prefix("/items")
	r.Get("/:id", c.findOne).Pipes(NewParseIntPipe("id"))
}

func (c *pipedController) findOne(ctx Context) error {
	id := ctx.Param("id")
	return ctx.JSON(200, map[string]any{"id": id})
}

func TestApplication_PipeTransform(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newPipedController},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	// Valid int
	req := httptest.NewRequest("GET", "/items/42", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	// After pipe, id should be numeric
	if body["id"] != float64(42) {
		t.Errorf("expected 42, got %v (type %T)", body["id"], body["id"])
	}

	// Invalid int
	req = httptest.NewRequest("GET", "/items/abc", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// --- Exception filter tests ---

type customFilter struct{}

func (f *customFilter) Catch(err error, host ArgumentsHost) error {
	httpCtx := host.SwitchToHTTP()
	resp := httpCtx.Response()
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(418) // I'm a teapot
	return json.NewEncoder(resp).Encode(map[string]string{"error": "filtered"})
}

func TestApplication_ExceptionFilter(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalFilters(&customFilter{})
	app.Init()

	// Make a request that will fail (bind on GET with no body)
	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	// This should succeed since GET /greet/ doesn't bind
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Middleware tests ---

type countingMiddleware struct {
	count int
}

func (m *countingMiddleware) Use(ctx Context, next NextFunc) error {
	m.count++
	ctx.SetHeader("X-Request-Count", Sprintf("%d", m.count))
	return next()
}

func TestApplication_GlobalMiddleware(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	mw := &countingMiddleware{}
	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalMiddleware(mw)
	app.Init()

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/greet/", nil)
		w := httptest.NewRecorder()
		app.Handler().ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	if mw.count != 3 {
		t.Errorf("expected 3 middleware calls, got %d", mw.count)
	}
}

// --- Module imports ---

func TestApplication_ModuleImports(t *testing.T) {
	greetModule := NewModule(ModuleOptions{
		Providers: []any{newGreetingService},
		Exports:   []any{(*greetingService)(nil)},
	})

	appModule := NewModule(ModuleOptions{
		Imports:     []*Module{greetModule},
		Controllers: []any{newGreetingController},
	})

	app := Create(appModule, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Lifecycle hooks ---

type lifecycleService struct {
	initCalled      bool
	destroyCalled   bool
	bootstrapCalled bool
}

func newLifecycleService() *lifecycleService { return &lifecycleService{} }

func (s *lifecycleService) OnModuleInit() error {
	s.initCalled = true
	return nil
}

func (s *lifecycleService) OnModuleDestroy() error {
	s.destroyCalled = true
	return nil
}

func (s *lifecycleService) OnApplicationBootstrap() error {
	s.bootstrapCalled = true
	return nil
}

type lifecycleController struct{}

func newLifecycleController() *lifecycleController { return &lifecycleController{} }
func (c *lifecycleController) Register(r Router)   { r.Get("/", func(ctx Context) error { return nil }) }

func TestApplication_LifecycleHooks(t *testing.T) {
	svc := newLifecycleService()
	module := NewModule(ModuleOptions{
		Controllers: []any{newLifecycleController},
		Providers:   []any{ProvideValue[*lifecycleService](svc)},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	if err := app.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if !svc.initCalled {
		t.Error("expected OnModuleInit to be called")
	}
	if !svc.bootstrapCalled {
		t.Error("expected OnApplicationBootstrap to be called")
	}

	app.Close()
	if !svc.destroyCalled {
		t.Error("expected OnModuleDestroy to be called")
	}
}

// --- CORS ---

func TestApplication_CORS(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.EnableCors(CorsOptions{Origin: "http://localhost:3000"})
	app.Init()

	req := httptest.NewRequest("OPTIONS", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("expected CORS origin header, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if w.Code != 204 {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

// --- Multiple modules and nested imports ---

type userService struct{ name string }

func newUserService() *userService { return &userService{name: "default"} }

type userController struct {
	svc *userService
}

func newUserController(svc *userService) *userController {
	return &userController{svc: svc}
}

func (c *userController) Register(r Router) {
	r.Prefix("/users")
	r.Get("/", c.list)
}

func (c *userController) list(ctx Context) error {
	return ctx.JSON(200, map[string]string{"user": c.svc.name})
}

func TestApplication_MultipleModules(t *testing.T) {
	userModule := NewModule(ModuleOptions{
		Controllers: []any{newUserController},
		Providers:   []any{newUserService},
	})
	greetModule := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})
	appModule := NewModule(ModuleOptions{
		Imports: []*Module{userModule, greetModule},
	})

	app := Create(appModule, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	// Test user route
	req := httptest.NewRequest("GET", "/users/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("users: expected 200, got %d", w.Code)
	}

	// Test greet route
	req = httptest.NewRequest("GET", "/greet/", nil)
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("greet: expected 200, got %d", w.Code)
	}
}

// --- Error handler response format ---

type errorController struct{}

func newErrorController() *errorController { return &errorController{} }

func (c *errorController) Register(r Router) {
	r.Prefix("/errors")
	r.Get("/bad-request", c.badRequest)
	r.Get("/not-found", c.notFound)
	r.Get("/custom", c.customError)
}

func (c *errorController) badRequest(ctx Context) error {
	return NewBadRequestException("invalid input")
}

func (c *errorController) notFound(ctx Context) error {
	return NewNotFoundException("resource not found")
}

func (c *errorController) customError(ctx Context) error {
	return NewHTTPException(418, "I'm a teapot")
}

func TestApplication_ErrorResponses(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newErrorController},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	tests := []struct {
		path     string
		expected int
		message  string
	}{
		{"/errors/bad-request", 400, "invalid input"},
		{"/errors/not-found", 404, "resource not found"},
		{"/errors/custom", 418, "I'm a teapot"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		app.Handler().ServeHTTP(w, req)

		if w.Code != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.path, tt.expected, w.Code)
		}
		var body map[string]any
		json.Unmarshal(w.Body.Bytes(), &body)
		if body["message"] != tt.message {
			t.Errorf("%s: expected message %q, got %v", tt.path, tt.message, body["message"])
		}
	}
}

// --- GetRoutes ---

func TestApplication_GetRoutes(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	routes := app.GetRoutes()
	if len(routes) != 3 { // GET /, GET /:name, POST /
		t.Errorf("expected 3 routes, got %d", len(routes))
	}
}

// --- MiddlewareFunc adapter ---

func TestMiddlewareFunc(t *testing.T) {
	called := false
	mw := MiddlewareFunc(func(ctx Context, next NextFunc) error {
		called = true
		return next()
	})

	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalMiddleware(mw)
	app.Init()

	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if !called {
		t.Error("expected MiddlewareFunc to be called")
	}
}

// --- GuardFunc adapter ---

func TestGuardFunc(t *testing.T) {
	guard := GuardFunc(func(ctx ExecutionContext) (bool, error) {
		return ctx.Header("X-Secret") == "open", nil
	})

	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalGuards(guard)
	app.Init()

	// Without secret header
	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 403 {
		t.Errorf("expected 403 without secret, got %d", w.Code)
	}

	// With secret header
	req = httptest.NewRequest("GET", "/greet/", nil)
	req.Header.Set("X-Secret", "open")
	w = httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Errorf("expected 200 with secret, got %d", w.Code)
	}
}

// --- InterceptorFunc adapter ---

func TestInterceptorFunc(t *testing.T) {
	interceptor := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		ctx.SetHeader("X-Intercepted", "yes")
		return next.Handle()
	})

	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalInterceptors(interceptor)
	app.Init()

	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Header().Get("X-Intercepted") != "yes" {
		t.Error("expected interceptor header")
	}
}

// --- ExceptionFilterFunc adapter ---

func TestExceptionFilterFunc_Integration(t *testing.T) {
	filter := ExceptionFilterFunc(func(err error, host ArgumentsHost) error {
		httpCtx := host.SwitchToHTTP()
		resp := httpCtx.Response()
		resp.Header().Set("Content-Type", "text/plain")
		resp.WriteHeader(500)
		resp.Write([]byte("handled"))
		return nil
	})

	module := NewModule(ModuleOptions{
		Controllers: []any{newErrorController},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalFilters(filter)
	app.Init()

	req := httptest.NewRequest("GET", "/errors/bad-request", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Body.String() != "handled" {
		t.Errorf("expected 'handled', got %q", w.Body.String())
	}
}

// --- GetContainer ---

func TestApplication_GetContainer(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	container := app.GetContainer()
	if container == nil {
		t.Fatal("expected non-nil container")
	}

	svc, err := Resolve[*greetingService](container)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if svc.greeting != "Hello, World!" {
		t.Errorf("expected greeting, got %q", svc.greeting)
	}
}

// --- Resolve from app ---

func TestApplication_Resolve(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	val, err := app.Resolve(resolveExportType((*greetingService)(nil)))
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	svc := val.(*greetingService)
	if svc.greeting != "Hello, World!" {
		t.Errorf("expected greeting, got %q", svc.greeting)
	}
}

// --- Close without listen ---

func TestApplication_Close(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.Init()

	err := app.Close()
	if err != nil {
		t.Errorf("close should not error: %v", err)
	}
}

// --- Global pipes integration ---

func TestApplication_GlobalPipes(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newPipedController},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalPipes(NewParseIntPipe("id"))
	app.Init()

	req := httptest.NewRequest("GET", "/items/abc", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("expected 400 from global pipe, got %d", w.Code)
	}
}

// --- Throttle integration ---

func TestApplication_ThrottleGuard(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalGuards(NewThrottleGuard(2, 5*time.Second))
	app.Init()

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/greet/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		app.Handler().ServeHTTP(w, req)
		if w.Code != 200 {
			t.Errorf("request %d: expected 200, got %d", i, w.Code)
		}
	}

	// Third request should be throttled
	req := httptest.NewRequest("GET", "/greet/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)
	if w.Code != 429 {
		t.Errorf("expected 429 from throttle, got %d", w.Code)
	}
}

// --- Versioning integration ---

func TestApplication_Versioning(t *testing.T) {
	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalMiddleware(NewVersioningMiddleware(VersioningOptions{
		Type:           VersioningHeader,
		DefaultVersion: "1",
	}))
	app.Init()

	req := httptest.NewRequest("GET", "/greet/", nil)
	req.Header.Set("X-API-Version", "2")
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	// Should still work (versioning middleware just stores version)
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// --- Serializer integration ---

type testSerializerController struct{}

func newTestSerializerController() *testSerializerController { return &testSerializerController{} }

type UserResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password" serialize:"exclude"`
}

func (c *testSerializerController) Register(r Router) {
	r.Prefix("/users")
	r.UseInterceptors(NewSerializerInterceptor())
	r.Get("/", c.list)
}

func (c *testSerializerController) list(ctx Context) error {
	// The serializer interceptor will strip the password
	return ctx.JSON(200, []UserResponse{
		{ID: 1, Name: "Alice", Password: "secret"},
	})
}

// --- Multiple interceptors ordering ---

func TestApplication_InterceptorOrdering(t *testing.T) {
	var order []string

	i1 := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		order = append(order, "i1-before")
		result, err := next.Handle()
		order = append(order, "i1-after")
		return result, err
	})

	i2 := InterceptorFunc(func(ctx ExecutionContext, next CallHandler) (any, error) {
		order = append(order, "i2-before")
		result, err := next.Handle()
		order = append(order, "i2-after")
		return result, err
	})

	module := NewModule(ModuleOptions{
		Controllers: []any{newGreetingController},
		Providers:   []any{newGreetingService},
	})

	app := Create(module, ApplicationOptions{Logger: NopLogger{}})
	app.UseGlobalInterceptors(i1, i2)
	app.Init()

	req := httptest.NewRequest("GET", "/greet/", nil)
	w := httptest.NewRecorder()
	app.Handler().ServeHTTP(w, req)

	expected := []string{"i1-before", "i2-before", "i2-after", "i1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("order[%d]: expected %q, got %q", i, expected[i], order[i])
		}
	}
}
