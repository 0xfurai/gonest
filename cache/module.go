package cache

import (
	"time"

	"github.com/gonest"
)

// Options configures the cache module.
type Options struct {
	TTL   time.Duration
	Store Store
}

// NewModule creates a cache module with memory store and configurable TTL.
func NewModule(opts ...Options) *gonest.Module {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	if opt.TTL == 0 {
		opt.TTL = 5 * time.Second
	}
	if opt.Store == nil {
		opt.Store = NewMemoryStore()
	}

	interceptor := NewCacheInterceptor(opt.Store, opt.TTL)

	return gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{
			gonest.ProvideValue[Store](opt.Store),
			gonest.ProvideValue[*CacheInterceptor](interceptor),
		},
		Exports: []any{
			gonest.ProvideValue[Store](opt.Store),
			gonest.ProvideValue[*CacheInterceptor](interceptor),
		},
	})
}
