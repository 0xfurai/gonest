package cache

import (
	"time"

	"github.com/gonest"
)

// CacheInterceptor caches GET request responses.
type CacheInterceptor struct {
	store Store
	ttl   time.Duration
}

// NewCacheInterceptor creates a cache interceptor with the given store and TTL.
func NewCacheInterceptor(store Store, ttl time.Duration) *CacheInterceptor {
	return &CacheInterceptor{store: store, ttl: ttl}
}

func (i *CacheInterceptor) Intercept(ctx gonest.ExecutionContext, next gonest.CallHandler) (any, error) {
	// Only cache GET requests
	if ctx.Method() != "GET" {
		return next.Handle()
	}

	key := ctx.Path() + "?" + ctx.Request().URL.RawQuery

	// Check cache
	if cached, ok := i.store.Get(key); ok {
		return cached, nil
	}

	// Execute handler
	result, err := next.Handle()
	if err != nil {
		return nil, err
	}

	// Store in cache
	i.store.Set(key, result, i.ttl)

	return result, nil
}
