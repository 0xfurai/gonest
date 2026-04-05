package gonest

// Guard determines whether a request should be handled by the route handler.
// Equivalent to NestJS CanActivate.
type Guard interface {
	CanActivate(ctx ExecutionContext) (bool, error)
}

// GuardFunc is a convenience adapter for simple guard functions.
type GuardFunc func(ctx ExecutionContext) (bool, error)

func (f GuardFunc) CanActivate(ctx ExecutionContext) (bool, error) {
	return f(ctx)
}
