package gonest

// Controller is the interface that all controllers must implement.
// Controllers register their routes via the Register method.
type Controller interface {
	Register(r Router)
}

// Router is passed to Controller.Register to define routes.
type Router interface {
	// Prefix sets the URL prefix for all routes in this controller.
	Prefix(path string)
	// Get registers a GET route.
	Get(path string, handler HandlerFunc) *RouteBuilder
	// Post registers a POST route.
	Post(path string, handler HandlerFunc) *RouteBuilder
	// Put registers a PUT route.
	Put(path string, handler HandlerFunc) *RouteBuilder
	// Delete registers a DELETE route.
	Delete(path string, handler HandlerFunc) *RouteBuilder
	// Patch registers a PATCH route.
	Patch(path string, handler HandlerFunc) *RouteBuilder
	// Options registers an OPTIONS route.
	Options(path string, handler HandlerFunc) *RouteBuilder
	// Head registers a HEAD route.
	Head(path string, handler HandlerFunc) *RouteBuilder
	// All registers a route for all HTTP methods.
	All(path string, handler HandlerFunc) *RouteBuilder
	// Search registers a SEARCH route (WebDAV).
	Search(path string, handler HandlerFunc) *RouteBuilder
	// Propfind registers a PROPFIND route (WebDAV).
	Propfind(path string, handler HandlerFunc) *RouteBuilder
	// Proppatch registers a PROPPATCH route (WebDAV).
	Proppatch(path string, handler HandlerFunc) *RouteBuilder
	// Mkcol registers a MKCOL route (WebDAV).
	Mkcol(path string, handler HandlerFunc) *RouteBuilder
	// Copy registers a COPY route (WebDAV).
	Copy(path string, handler HandlerFunc) *RouteBuilder
	// Move registers a MOVE route (WebDAV).
	Move(path string, handler HandlerFunc) *RouteBuilder
	// Lock registers a LOCK route (WebDAV).
	Lock(path string, handler HandlerFunc) *RouteBuilder
	// Unlock registers an UNLOCK route (WebDAV).
	Unlock(path string, handler HandlerFunc) *RouteBuilder
	// UseGuards applies guards to all routes in this controller.
	UseGuards(guards ...any)
	// UseInterceptors applies interceptors to all routes in this controller.
	UseInterceptors(interceptors ...any)
	// UsePipes applies pipes to all routes in this controller.
	UsePipes(pipes ...Pipe)
	// UseFilters applies exception filters to all routes in this controller.
	UseFilters(filters ...ExceptionFilter)
}

// MiddlewareConfigurer allows modules to configure middleware.
// Modules that need middleware should implement this interface.
type MiddlewareConfigurer interface {
	Configure(consumer MiddlewareConsumer)
}

// MiddlewareConsumer is used to apply middleware to routes.
type MiddlewareConsumer interface {
	Apply(middleware ...Middleware) MiddlewareConsumerRoutes
}

// MiddlewareConsumerRoutes specifies which routes middleware applies to.
type MiddlewareConsumerRoutes interface {
	ForRoutes(routes ...string) MiddlewareConsumer
	Exclude(routes ...string) MiddlewareConsumerRoutes
}
