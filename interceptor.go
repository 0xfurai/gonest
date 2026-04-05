package gonest

// Interceptor wraps the execution of a route handler, enabling logic before
// and after the handler runs. Equivalent to NestJS NestInterceptor.
type Interceptor interface {
	Intercept(ctx ExecutionContext, next CallHandler) (any, error)
}

// InterceptorFunc is a convenience adapter for simple interceptor functions.
type InterceptorFunc func(ctx ExecutionContext, next CallHandler) (any, error)

func (f InterceptorFunc) Intercept(ctx ExecutionContext, next CallHandler) (any, error) {
	return f(ctx, next)
}
