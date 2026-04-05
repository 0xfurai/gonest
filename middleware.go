package gonest

// Middleware processes HTTP requests before they reach the route handler.
// Equivalent to NestJS NestMiddleware.
type Middleware interface {
	Use(ctx Context, next NextFunc) error
}

// MiddlewareFunc is a convenience adapter for function-based middleware.
type MiddlewareFunc func(ctx Context, next NextFunc) error

func (f MiddlewareFunc) Use(ctx Context, next NextFunc) error {
	return f(ctx, next)
}
