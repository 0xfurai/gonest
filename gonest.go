package gonest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"syscall"

	"github.com/0xfurai/gonest/platform"
	"github.com/0xfurai/gonest/platform/stdhttp"
)

// Application is the main application instance, equivalent to NestJS INestApplication.
type Application struct {
	module     *Module
	adapter    platform.HTTPAdapter
	logger     Logger
	reflector  *Reflector
	routes     []*Route

	globalGuards       []any
	globalInterceptors []any
	globalPipes        []Pipe
	globalFilters      []ExceptionFilter
	globalMiddleware   []Middleware
	globalPrefix       string

	discovery      *DiscoveryService
	graphInspector *GraphInspector
	lazyLoader     *LazyModuleLoader

	shutdownSignals []os.Signal
	globalPrefixExcludes []string
	viewEngine     ViewEngine
	sessionStore   SessionStore
}

// ApplicationOptions configures the application.
type ApplicationOptions struct {
	// Logger overrides the default logger.
	Logger Logger
	// Adapter overrides the default HTTP adapter.
	Adapter platform.HTTPAdapter
}

// Create bootstraps the application from the root module.
func Create(rootModule *Module, opts ...ApplicationOptions) *Application {
	var opt ApplicationOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	logger := opt.Logger
	if logger == nil {
		logger = NewDefaultLogger()
	}

	adapter := opt.Adapter
	if adapter == nil {
		adapter = stdhttp.New()
	}

	reflector := NewReflector()
	app := &Application{
		module:         rootModule,
		adapter:        adapter,
		logger:         logger,
		reflector:      reflector,
		discovery:      NewDiscoveryService(reflector),
		graphInspector: NewGraphInspector(),
	}

	return app
}

// UseGlobalGuards registers guards that apply to every route.
func (app *Application) UseGlobalGuards(guards ...any) *Application {
	app.globalGuards = append(app.globalGuards, guards...)
	return app
}

// UseGlobalInterceptors registers interceptors that apply to every route.
func (app *Application) UseGlobalInterceptors(interceptors ...any) *Application {
	app.globalInterceptors = append(app.globalInterceptors, interceptors...)
	return app
}

// UseGlobalPipes registers pipes that apply to every route.
func (app *Application) UseGlobalPipes(pipes ...Pipe) *Application {
	app.globalPipes = append(app.globalPipes, pipes...)
	return app
}

// UseGlobalFilters registers exception filters that apply to every route.
func (app *Application) UseGlobalFilters(filters ...ExceptionFilter) *Application {
	app.globalFilters = append(app.globalFilters, filters...)
	return app
}

// UseGlobalMiddleware registers middleware that runs on every request.
func (app *Application) UseGlobalMiddleware(middleware ...Middleware) *Application {
	app.globalMiddleware = append(app.globalMiddleware, middleware...)
	return app
}

// SetGlobalPrefix sets a prefix for all routes (e.g., "/api").
func (app *Application) SetGlobalPrefix(prefix string) *Application {
	app.globalPrefix = prefix
	return app
}

// EnableCors enables CORS with the given options.
func (app *Application) EnableCors(opts ...CorsOptions) *Application {
	var corsOpt CorsOptions
	if len(opts) > 0 {
		corsOpt = opts[0]
	} else {
		corsOpt = CorsOptions{Origin: "*"}
	}
	app.adapter.Use(corsMiddleware(corsOpt))
	return app
}

// Listen compiles the module tree, registers all routes, and starts the HTTP server.
func (app *Application) Listen(addr string) error {
	if err := app.init(); err != nil {
		return err
	}
	app.logger.Log("Application is running on: http://localhost%s", addr)
	return app.adapter.Listen(addr)
}

// Init compiles the module tree and registers routes without starting the server.
// Useful for testing.
func (app *Application) Init() error {
	return app.init()
}

// Handler returns the underlying http.Handler for testing with httptest.
func (app *Application) Handler() http.Handler {
	return app.adapter.Handler()
}

// Close gracefully shuts down the application.
func (app *Application) Close() error {
	// Run shutdown hooks
	for _, mod := range app.module.allModules() {
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(BeforeApplicationShutdown); ok {
				_ = hook.BeforeApplicationShutdown("")
			}
		}
	}

	err := app.adapter.Shutdown()

	for _, mod := range app.module.allModules() {
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(OnApplicationShutdown); ok {
				_ = hook.OnApplicationShutdown("")
			}
		}
	}

	_ = app.module.destroy()
	return err
}

// ListenAndServeWithGracefulShutdown starts the server and handles OS signals.
func (app *Application) ListenAndServeWithGracefulShutdown(addr string) error {
	if err := app.init(); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		app.logger.Log("Application is running on: http://localhost%s", addr)
		errCh <- app.adapter.Listen(addr)
	}()

	quit := make(chan os.Signal, 1)
	signals := app.shutdownSignals
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	signal.Notify(quit, signals...)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		app.logger.Log("Received signal %v, shutting down...", sig)
		return app.CloseWithSignal(sig.String())
	}
}

// CloseWithSignal gracefully shuts down the application, passing the signal name
// to shutdown hooks.
func (app *Application) CloseWithSignal(signal string) error {
	for _, mod := range app.module.allModules() {
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(BeforeApplicationShutdown); ok {
				_ = hook.BeforeApplicationShutdown(signal)
			}
		}
	}

	err := app.adapter.Shutdown()

	for _, mod := range app.module.allModules() {
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(OnApplicationShutdown); ok {
				_ = hook.OnApplicationShutdown(signal)
			}
		}
	}

	_ = app.module.destroy()
	return err
}

// GetContainer returns the root module's DI container.
func (app *Application) GetContainer() *Container {
	return app.module.container
}

// GetDiscoveryService returns the discovery service for runtime introspection.
func (app *Application) GetDiscoveryService() *DiscoveryService {
	return app.discovery
}

// GetGraphInspector returns the graph inspector for dependency graph analysis.
func (app *Application) GetGraphInspector() *GraphInspector {
	return app.graphInspector
}

// GetLazyModuleLoader returns the lazy module loader for on-demand module loading.
func (app *Application) GetLazyModuleLoader() *LazyModuleLoader {
	if app.lazyLoader == nil {
		app.lazyLoader = NewLazyModuleLoader(app)
	}
	return app.lazyLoader
}

// EnableShutdownHooks enables graceful shutdown with specific signals.
// If no signals are provided, defaults to SIGINT and SIGTERM.
func (app *Application) EnableShutdownHooks(signals ...os.Signal) *Application {
	if len(signals) == 0 {
		signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}
	app.shutdownSignals = signals
	return app
}

// SetGlobalPrefixWithExclude sets a prefix for all routes with exclusion patterns.
func (app *Application) SetGlobalPrefixWithExclude(prefix string, excludes ...string) *Application {
	app.globalPrefix = prefix
	app.globalPrefixExcludes = excludes
	return app
}

// SetViewEngine sets the template rendering engine.
func (app *Application) SetViewEngine(engine ViewEngine) *Application {
	app.viewEngine = engine
	return app
}

// SetSessionStore sets the session store for @Session support.
func (app *Application) SetSessionStore(store SessionStore) *Application {
	app.sessionStore = store
	return app
}

func (app *Application) init() error {
	// Compile the module tree
	if err := app.module.compile(nil, app.logger, app.reflector); err != nil {
		return fmt.Errorf("gonest: module compilation failed: %w", err)
	}

	// Collect all controllers and register their routes
	controllers := app.module.allControllers()
	for _, ctrl := range controllers {
		router := newRouter()
		ctrl.Register(router)
		routes := router.resolvedRoutes()

		// If controller implements HostController, add host middleware to routes
		if hc, ok := ctrl.(HostController); ok {
			hostMW := newHostMatchMiddleware(hc.Host())
			for _, route := range routes {
				existing := route.Guards
				route.Guards = append([]any{hostGuardAdapter{mw: hostMW}}, existing...)
			}
		}

		for _, route := range routes {
			app.registerRoute(route, ctrl)
		}
	}

	// Configure middleware from modules
	for _, mod := range app.module.allModules() {
		// Check if any provider implements MiddlewareConfigurer
		if mod.container == nil {
			continue
		}
		providers, _ := mod.container.ResolveAll()
		for _, p := range providers {
			if mc, ok := p.(MiddlewareConfigurer); ok {
				consumer := newMiddlewareConsumer(app)
				mc.Configure(consumer)
			}
		}
	}

	// Run OnApplicationBootstrap hooks
	for _, mod := range app.module.allModules() {
		if mod.container == nil {
			continue
		}
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			if hook, ok := inst.(OnApplicationBootstrap); ok {
				if err := hook.OnApplicationBootstrap(); err != nil {
					return err
				}
			}
		}
	}

	// Populate discovery service and graph inspector
	allMods := app.module.allModules()
	app.discovery.SetModules(allMods)
	app.graphInspector.SetModules(allMods)

	// Register DiscoveryService and GraphInspector in root container
	if app.module.container != nil {
		app.module.container.RegisterInstance(
			reflect.TypeOf((*DiscoveryService)(nil)), app.discovery)
		app.module.container.RegisterInstance(
			reflect.TypeOf((*GraphInspector)(nil)), app.graphInspector)
	}

	// Feed registered routes to any RouteConsumer providers (e.g., swagger generator)
	app.feedRouteConsumers()

	return nil
}

// RouteConsumer is implemented by providers that want to receive all registered
// routes after initialization (e.g., swagger documentation generators).
type RouteConsumer interface {
	ConsumeRoute(method, path string, metadata map[string]any)
}

func (app *Application) feedRouteConsumers() {
	for _, mod := range app.module.allModules() {
		if mod.container == nil {
			continue
		}
		instances, _ := mod.container.ResolveAll()
		for _, inst := range instances {
			consumer, ok := inst.(RouteConsumer)
			if !ok {
				continue
			}
			for _, route := range app.routes {
				path := route.Path
				if !app.isExcludedFromPrefix(route.Path) {
					path = app.globalPrefix + route.Path
				}
				consumer.ConsumeRoute(route.Method, path, route.Metadata)
			}
		}
	}
}

func (app *Application) registerRoute(route *Route, ctrl Controller) {
	path := route.Path
	// Apply global prefix unless the route is excluded
	if !app.isExcludedFromPrefix(route.Path) {
		path = app.globalPrefix + route.Path
	}
	if path == "" {
		path = "/"
	}

	// Store metadata in reflector for guard access
	handlerID := fmt.Sprintf("%p", route.Handler)
	for k, v := range route.Metadata {
		app.reflector.Set(handlerID, k, v)
	}

	handler := func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		ctx := newContext(w, r)

		// Set path params
		for k, v := range params {
			ctx.setParam(k, v)
		}

		// Build execution context
		execCtx := newExecutionContext(ctx, handlerID, ctrl, route.Metadata)

		// Execute the pipeline
		err := app.executePipeline(execCtx, route, ctx)
		if err != nil && !ctx.Written() {
			app.handleError(err, ctx)
		}
	}

	if route.Method == "*" {
		for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"} {
			app.adapter.Handle(m, path, handler)
		}
	} else {
		app.adapter.Handle(route.Method, path, handler)
	}

	app.routes = append(app.routes, route)
	app.logger.Log("Mapped {%s %s} route", route.Method, path)
}

func (app *Application) executePipeline(execCtx *executionContext, route *Route, ctx *defaultContext) error {
	// Inject view engine if configured
	if app.viewEngine != nil {
		ctx.Set("__view_engine", app.viewEngine)
	}

	// 1. Run global middleware
	middlewareChain := append(app.globalMiddleware[:0:0], app.globalMiddleware...)

	return app.runMiddleware(middlewareChain, 0, ctx, func() error {
		// 2. Run guards (global, then route-level)
		allGuards := append(app.globalGuards[:0:0], app.globalGuards...)
		allGuards = append(allGuards, route.Guards...)

		for _, g := range allGuards {
			guard := app.resolveGuard(g)
			if guard == nil {
				continue
			}
			allowed, err := guard.CanActivate(execCtx)
			if err != nil {
				return err
			}
			if !allowed {
				return NewForbiddenException("Forbidden resource")
			}
		}

		// 3. Run interceptors + pipes + handler
		allInterceptors := append(app.globalInterceptors[:0:0], app.globalInterceptors...)
		allInterceptors = append(allInterceptors, route.Interceptors...)

		return app.runInterceptors(allInterceptors, 0, execCtx, func() (any, error) {
			// 4. Run pipes on params
			allPipes := append(app.globalPipes[:0:0], app.globalPipes...)
			allPipes = append(allPipes, route.Pipes...)

			for _, pipe := range allPipes {
				// Apply pipes to all path params
				for name, val := range ctx.params {
					meta := ArgumentMetadata{Type: "param", Name: name}
					transformed, err := pipe.Transform(val, meta)
					if err != nil {
						return nil, err
					}
					ctx.setParam(name, transformed)
				}
			}

			// 5. Set response headers from metadata
			if headers, ok := route.Metadata["__headers"].([][2]string); ok {
				for _, h := range headers {
					ctx.SetHeader(h[0], h[1])
				}
			}

			// 6. Handle redirect
			if redirect, ok := route.Metadata["__redirect"].([2]any); ok {
				url := redirect[0].(string)
				code := redirect[1].(int)
				return nil, ctx.Redirect(code, url)
			}

			// 7. Execute handler
			err := route.Handler(ctx)
			return nil, err
		})
	})
}

func (app *Application) runMiddleware(mw []Middleware, idx int, ctx Context, final NextFunc) error {
	if idx >= len(mw) {
		return final()
	}
	return mw[idx].Use(ctx, func() error {
		return app.runMiddleware(mw, idx+1, ctx, final)
	})
}

func (app *Application) runInterceptors(interceptors []any, idx int, ctx ExecutionContext, handler func() (any, error)) error {
	if idx >= len(interceptors) {
		_, err := handler()
		return err
	}

	interceptor := app.resolveInterceptor(interceptors[idx])
	if interceptor == nil {
		return app.runInterceptors(interceptors, idx+1, ctx, handler)
	}

	next := NewCallHandler(func() (any, error) {
		var finalResult any
		var finalErr error
		err := app.runInterceptors(interceptors, idx+1, ctx, func() (any, error) {
			result, err := handler()
			finalResult = result
			finalErr = err
			return result, err
		})
		if err != nil {
			return nil, err
		}
		return finalResult, finalErr
	})

	_, err := interceptor.Intercept(ctx, next)
	return err
}

func (app *Application) resolveGuard(g any) Guard {
	if guard, ok := g.(Guard); ok {
		return guard
	}
	// Try to resolve from DI container (constructor function)
	if app.module.container != nil {
		ft := reflect.TypeOf(g)
		if ft != nil && ft.Kind() == reflect.Func {
			p := Provide(g)
			app.module.container.Register(p)
			instance, err := app.module.container.Resolve(p.Type)
			if err == nil {
				if guard, ok := instance.(Guard); ok {
					return guard
				}
			}
		}
	}
	return nil
}

func (app *Application) resolveInterceptor(i any) Interceptor {
	if interceptor, ok := i.(Interceptor); ok {
		return interceptor
	}
	// Try to resolve from DI container (constructor function)
	if app.module.container != nil {
		ft := reflect.TypeOf(i)
		if ft != nil && ft.Kind() == reflect.Func {
			p := Provide(i)
			app.module.container.Register(p)
			instance, err := app.module.container.Resolve(p.Type)
			if err == nil {
				if interceptor, ok := instance.(Interceptor); ok {
					return interceptor
				}
			}
		}
	}
	return nil
}

func (app *Application) handleError(err error, ctx *defaultContext) {
	// Try route-level and global filters
	filters := append([]ExceptionFilter{}, app.globalFilters...)

	host := newArgumentsHost(ctx)
	for _, f := range filters {
		if filterErr := f.Catch(err, host); filterErr == nil {
			return
		}
	}

	// Default filter
	defaultFilter := &DefaultExceptionFilter{}
	_ = defaultFilter.Catch(err, host)
}

// CorsOptions configures CORS behavior.
type CorsOptions struct {
	Origin      string
	Methods     string
	Headers     string
	Credentials bool
}

func corsMiddleware(opts CorsOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := opts.Origin
			if origin == "" {
				origin = "*"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)

			methods := opts.Methods
			if methods == "" {
				methods = "GET, POST, PUT, DELETE, PATCH, OPTIONS"
			}
			w.Header().Set("Access-Control-Allow-Methods", methods)

			headers := opts.Headers
			if headers == "" {
				headers = "Content-Type, Authorization"
			}
			w.Header().Set("Access-Control-Allow-Headers", headers)

			if opts.Credentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// middlewareConsumerImpl implements MiddlewareConsumer for module middleware configuration.
type middlewareConsumerImpl struct {
	app *Application
}

func newMiddlewareConsumer(app *Application) MiddlewareConsumer {
	return &middlewareConsumerImpl{app: app}
}

func (mc *middlewareConsumerImpl) Apply(middleware ...Middleware) MiddlewareConsumerRoutes {
	return &middlewareRoutesImpl{
		app:        mc.app,
		middleware: middleware,
		consumer:   mc,
	}
}

type middlewareRoutesImpl struct {
	app        *Application
	middleware []Middleware
	consumer   MiddlewareConsumer
	excludes   []string
}

func (mr *middlewareRoutesImpl) ForRoutes(routes ...string) MiddlewareConsumer {
	for _, mw := range mr.middleware {
		mr.app.globalMiddleware = append(mr.app.globalMiddleware, &routeFilteredMiddleware{
			inner:    mw,
			routes:   routes,
			excludes: mr.excludes,
		})
	}
	return mr.consumer
}

func (mr *middlewareRoutesImpl) Exclude(routes ...string) MiddlewareConsumerRoutes {
	mr.excludes = append(mr.excludes, routes...)
	return mr
}

type routeFilteredMiddleware struct {
	inner    Middleware
	routes   []string
	excludes []string
}

func (m *routeFilteredMiddleware) Use(ctx Context, next NextFunc) error {
	path := ctx.Path()

	// Check excludes
	for _, ex := range m.excludes {
		if matchRoute(ex, path) {
			return next()
		}
	}

	// Check includes
	for _, route := range m.routes {
		if matchRoute(route, path) {
			return m.inner.Use(ctx, next)
		}
	}

	// If routes is empty, match all
	if len(m.routes) == 0 {
		return m.inner.Use(ctx, next)
	}

	return next()
}

func matchRoute(pattern, path string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}
	return pattern == path
}

// isExcludedFromPrefix checks if a route path is excluded from the global prefix.
func (app *Application) isExcludedFromPrefix(path string) bool {
	for _, exclude := range app.globalPrefixExcludes {
		if matchRoute(exclude, path) {
			return true
		}
	}
	return false
}

// GetRoutes returns all registered routes for inspection.
func (app *Application) GetRoutes() []*Route {
	return app.routes
}

// Resolve is a convenience to resolve a type from the root container.
func (app *Application) Resolve(t reflect.Type) (any, error) {
	return app.module.container.Resolve(t)
}

// ServeHTTP serializes the error response for unhandled JSON errors.
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"statusCode": statusCode,
		"message":    message,
	})
}
