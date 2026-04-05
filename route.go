package gonest

import "net/http"

// Route holds all the information about a single route.
type Route struct {
	Method       string
	Path         string
	Handler      HandlerFunc
	Metadata     map[string]any
	Guards       []any // Guard constructors or instances
	Interceptors []any // Interceptor constructors or instances
	Pipes        []Pipe
	Filters      []ExceptionFilter
}

// RouteBuilder provides a fluent API for configuring individual routes.
type RouteBuilder struct {
	route  *Route
	router *routerImpl
}

// SetMetadata attaches metadata to this route, readable by guards/interceptors.
func (rb *RouteBuilder) SetMetadata(key string, value any) *RouteBuilder {
	rb.route.Metadata[key] = value
	return rb
}

// Guards applies guards to this specific route.
func (rb *RouteBuilder) Guards(guards ...any) *RouteBuilder {
	rb.route.Guards = append(rb.route.Guards, guards...)
	return rb
}

// Interceptors applies interceptors to this specific route.
func (rb *RouteBuilder) Interceptors(interceptors ...any) *RouteBuilder {
	rb.route.Interceptors = append(rb.route.Interceptors, interceptors...)
	return rb
}

// Pipes applies pipes to this specific route.
func (rb *RouteBuilder) Pipes(pipes ...Pipe) *RouteBuilder {
	rb.route.Pipes = append(rb.route.Pipes, pipes...)
	return rb
}

// Filters applies exception filters to this specific route.
func (rb *RouteBuilder) Filters(filters ...ExceptionFilter) *RouteBuilder {
	rb.route.Filters = append(rb.route.Filters, filters...)
	return rb
}

// Summary sets a short description for this route (used by swagger).
func (rb *RouteBuilder) Summary(text string) *RouteBuilder {
	rb.route.Metadata["summary"] = text
	return rb
}

// Tags sets swagger tags for grouping this route.
func (rb *RouteBuilder) Tags(tags ...string) *RouteBuilder {
	rb.route.Metadata["tags"] = tags
	return rb
}

// Body declares the request body type for documentation.
// Pass a zero-value struct: .Body(CreateCatDto{})
func (rb *RouteBuilder) Body(example any) *RouteBuilder {
	rb.route.Metadata["__body"] = example
	return rb
}

// Response declares the success response type and status code for documentation.
// Pass a zero-value struct or slice: .Response(200, []Cat{})
func (rb *RouteBuilder) Response(statusCode int, example any) *RouteBuilder {
	rb.route.Metadata["__responseType"] = example
	if statusCode > 0 {
		rb.route.Metadata["__httpCode"] = statusCode
	}
	return rb
}

// HttpCode sets the default HTTP status code for a successful response.
func (rb *RouteBuilder) HttpCode(code int) *RouteBuilder {
	rb.route.Metadata["__httpCode"] = code
	return rb
}

// Header adds a response header to this route.
func (rb *RouteBuilder) Header(name, value string) *RouteBuilder {
	headers, ok := rb.route.Metadata["__headers"].([][2]string)
	if !ok {
		headers = nil
	}
	headers = append(headers, [2]string{name, value})
	rb.route.Metadata["__headers"] = headers
	return rb
}

// Redirect configures a redirect for this route.
func (rb *RouteBuilder) Redirect(url string, statusCode int) *RouteBuilder {
	rb.route.Metadata["__redirect"] = [2]any{url, statusCode}
	return rb
}

// Render sets the template to render for this route.
// Equivalent to NestJS @Render() decorator.
func (rb *RouteBuilder) Render(template string) *RouteBuilder {
	rb.route.Metadata["render"] = template
	return rb
}

// routerImpl is the default Router implementation used during controller registration.
type routerImpl struct {
	prefix             string
	routes             []*Route
	controllerGuards   []any
	controllerInterceptors []any
	controllerPipes    []Pipe
	controllerFilters  []ExceptionFilter
}

func newRouter() *routerImpl {
	return &routerImpl{}
}

func (r *routerImpl) Prefix(path string) {
	r.prefix = path
}

func (r *routerImpl) addRoute(method, path string, handler HandlerFunc) *RouteBuilder {
	route := &Route{
		Method:   method,
		Path:     r.prefix + path,
		Handler:  handler,
		Metadata: make(map[string]any),
	}
	r.routes = append(r.routes, route)
	return &RouteBuilder{route: route, router: r}
}

func (r *routerImpl) Get(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute(http.MethodGet, path, handler)
}

func (r *routerImpl) Post(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute(http.MethodPost, path, handler)
}

func (r *routerImpl) Put(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute(http.MethodPut, path, handler)
}

func (r *routerImpl) Delete(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute(http.MethodDelete, path, handler)
}

func (r *routerImpl) Patch(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute(http.MethodPatch, path, handler)
}

func (r *routerImpl) Options(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute(http.MethodOptions, path, handler)
}

func (r *routerImpl) Head(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute(http.MethodHead, path, handler)
}

func (r *routerImpl) All(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("*", path, handler)
}

func (r *routerImpl) Search(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("SEARCH", path, handler)
}

func (r *routerImpl) Propfind(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("PROPFIND", path, handler)
}

func (r *routerImpl) Proppatch(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("PROPPATCH", path, handler)
}

func (r *routerImpl) Mkcol(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("MKCOL", path, handler)
}

func (r *routerImpl) Copy(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("COPY", path, handler)
}

func (r *routerImpl) Move(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("MOVE", path, handler)
}

func (r *routerImpl) Lock(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("LOCK", path, handler)
}

func (r *routerImpl) Unlock(path string, handler HandlerFunc) *RouteBuilder {
	return r.addRoute("UNLOCK", path, handler)
}

func (r *routerImpl) UseGuards(guards ...any) {
	r.controllerGuards = append(r.controllerGuards, guards...)
}

func (r *routerImpl) UseInterceptors(interceptors ...any) {
	r.controllerInterceptors = append(r.controllerInterceptors, interceptors...)
}

func (r *routerImpl) UsePipes(pipes ...Pipe) {
	r.controllerPipes = append(r.controllerPipes, pipes...)
}

func (r *routerImpl) UseFilters(filters ...ExceptionFilter) {
	r.controllerFilters = append(r.controllerFilters, filters...)
}

// resolvedRoutes returns all routes with controller-level guards/interceptors/pipes merged.
func (r *routerImpl) resolvedRoutes() []*Route {
	for _, route := range r.routes {
		route.Guards = append(r.controllerGuards, route.Guards...)
		route.Interceptors = append(r.controllerInterceptors, route.Interceptors...)
		route.Pipes = append(r.controllerPipes, route.Pipes...)
		route.Filters = append(r.controllerFilters, route.Filters...)
	}
	return r.routes
}
