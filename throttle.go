package gonest

import (
	"sync"
	"time"
)

// ThrottleGuard implements rate limiting using the token bucket algorithm.
// Apply globally or per-route to limit request frequency.
type ThrottleGuard struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int           // max requests per window
	window  time.Duration // time window
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

// NewThrottleGuard creates a rate-limiting guard.
// limit: max requests allowed within the window duration.
func NewThrottleGuard(limit int, window time.Duration) *ThrottleGuard {
	return &ThrottleGuard{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
	}
}

func (g *ThrottleGuard) CanActivate(ctx ExecutionContext) (bool, error) {
	key := g.getKey(ctx)

	g.mu.Lock()
	defer g.mu.Unlock()

	b, ok := g.buckets[key]
	now := time.Now()
	if !ok {
		g.buckets[key] = &bucket{tokens: g.limit - 1, lastReset: now}
		return true, nil
	}

	// Reset bucket if window has passed
	if now.Sub(b.lastReset) >= g.window {
		b.tokens = g.limit - 1
		b.lastReset = now
		return true, nil
	}

	if b.tokens <= 0 {
		return false, NewTooManyRequestsException("rate limit exceeded")
	}

	b.tokens--
	return true, nil
}

func (g *ThrottleGuard) getKey(ctx ExecutionContext) string {
	return ctx.IP()
}

// ThrottleByMetadata allows configuring per-route rate limits via metadata.
// Set metadata "throttle_limit" and "throttle_window" on routes.
type ThrottleByMetadataGuard struct {
	mu          sync.Mutex
	buckets     map[string]*bucket
	defaultLimit  int
	defaultWindow time.Duration
}

func NewThrottleByMetadataGuard(defaultLimit int, defaultWindow time.Duration) *ThrottleByMetadataGuard {
	return &ThrottleByMetadataGuard{
		buckets:       make(map[string]*bucket),
		defaultLimit:  defaultLimit,
		defaultWindow: defaultWindow,
	}
}

func (g *ThrottleByMetadataGuard) CanActivate(ctx ExecutionContext) (bool, error) {
	limit := g.defaultLimit
	window := g.defaultWindow

	if l, ok := GetMetadata[int](ctx, "throttle_limit"); ok {
		limit = l
	}
	if w, ok := GetMetadata[time.Duration](ctx, "throttle_window"); ok {
		window = w
	}

	key := ctx.IP() + ":" + ctx.Path()

	g.mu.Lock()
	defer g.mu.Unlock()

	b, ok := g.buckets[key]
	now := time.Now()
	if !ok {
		g.buckets[key] = &bucket{tokens: limit - 1, lastReset: now}
		return true, nil
	}

	if now.Sub(b.lastReset) >= window {
		b.tokens = limit - 1
		b.lastReset = now
		return true, nil
	}

	if b.tokens <= 0 {
		return false, NewTooManyRequestsException("rate limit exceeded")
	}

	b.tokens--
	return true, nil
}
