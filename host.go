package gonest

import (
	"net/http"
	"strings"
)

// HostController is an interface for controllers that are bound to a specific
// host pattern, enabling subdomain routing. Equivalent to NestJS
// @Controller({ host: ':tenant.example.com' }).
//
// Usage:
//
//	type TenantController struct{}
//	func (c *TenantController) Host() string { return ":tenant.example.com" }
//	func (c *TenantController) Register(r gonest.Router) {
//	    r.Get("/info", func(ctx gonest.Context) error {
//	        tenant := gonest.HostParam(ctx, "tenant")
//	        return ctx.JSON(200, map[string]string{"tenant": tenant})
//	    })
//	}
type HostController interface {
	Controller
	// Host returns the host pattern, e.g., ":tenant.example.com".
	Host() string
}

// HostParam retrieves a host parameter extracted from subdomain routing.
func HostParam(ctx Context, name string) string {
	val, ok := ctx.Get("__host_" + name)
	if !ok {
		return ""
	}
	s, _ := val.(string)
	return s
}

// GetHostParams retrieves all host parameters from the context.
func GetHostParams(ctx Context) map[string]string {
	val, ok := ctx.Get("__host_params")
	if !ok {
		return nil
	}
	m, _ := val.(map[string]string)
	return m
}

// hostMatchMiddleware checks if the request host matches the controller's host pattern.
// If matched, it extracts host parameters (e.g., subdomain) into the context.
type hostMatchMiddleware struct {
	pattern string
}

func newHostMatchMiddleware(pattern string) *hostMatchMiddleware {
	return &hostMatchMiddleware{pattern: pattern}
}

func (m *hostMatchMiddleware) Use(ctx Context, next NextFunc) error {
	host := extractHost(ctx.Request())
	params, ok := matchHost(m.pattern, host)
	if !ok {
		return NewNotFoundException("Cannot " + ctx.Method() + " " + ctx.Path())
	}
	ctx.Set("__host_params", params)
	for k, v := range params {
		ctx.Set("__host_"+k, v)
	}
	return next()
}

// hostGuardAdapter wraps host matching as a guard so it runs in the pipeline.
type hostGuardAdapter struct {
	mw *hostMatchMiddleware
}

func (g hostGuardAdapter) CanActivate(ctx ExecutionContext) (bool, error) {
	host := extractHost(ctx.Request())
	params, ok := matchHost(g.mw.pattern, host)
	if !ok {
		return false, nil
	}
	ctx.Set("__host_params", params)
	for k, v := range params {
		ctx.Set("__host_"+k, v)
	}
	return true, nil
}

// extractHost returns the host from the request, stripping the port.
func extractHost(r *http.Request) string {
	host := r.Host
	if i := strings.LastIndex(host, ":"); i >= 0 {
		// Check if it's a port (not an IPv6 address)
		if !strings.Contains(host[i:], "]") {
			host = host[:i]
		}
	}
	return host
}

// matchHost matches a host against a pattern like ":tenant.example.com".
// Segments prefixed with ":" are treated as parameters.
// Returns the extracted parameters and whether the match succeeded.
func matchHost(pattern, host string) (map[string]string, bool) {
	patternParts := strings.Split(pattern, ".")
	hostParts := strings.Split(host, ".")

	if len(patternParts) != len(hostParts) {
		return nil, false
	}

	params := make(map[string]string)
	for i, pp := range patternParts {
		if strings.HasPrefix(pp, ":") {
			params[pp[1:]] = hostParts[i]
		} else if pp != hostParts[i] {
			return nil, false
		}
	}
	return params, true
}
